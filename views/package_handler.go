package views

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"math"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Khryzen/esl_system/models"
	"github.com/Khryzen/esl_system/utils"
	"github.com/uadmin/uadmin"
)

const (
	packageDateLayout = "2006-01-02"
	// Largest image accepted (5 MB).
	maxPackageImageSize = 5 << 20
)

// Allowed image types, detected from the file's bytes (not its name).
var packageImageTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

type packageInputError string

func (e packageInputError) Error() string { return string(e) }

// PackageHandler serves the Packages page and its Add / Edit requests.
//
//	GET  /package/          render the page (AllPackages)
//	POST /package/          create a package (multipart form)
//	PUT  /package/?id=<id>  update a package (multipart form)
func PackageHandler(w http.ResponseWriter, r *http.Request) map[string]interface{} {
	context := map[string]interface{}{}

	switch r.Method {
	case http.MethodPost:
		savePackage(w, r, false)
		return context
	case http.MethodPut:
		savePackage(w, r, true)
		return context
	}

	packages := []models.Package{}
	uadmin.All(&packages)

	for i := range packages {
		if packages[i].Image == "" {
			continue
		}
		if url, err := utils.GetPresignedFileURL(filepath.Base(packages[i].Image)); err == nil {
			packages[i].Image = url
		}
	}
	context["AllPackages"] = packages

	return context
}

func savePackage(w http.ResponseWriter, r *http.Request, isEdit bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxPackageImageSize+(1<<20))
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		packageFail(w, r, packageInputError("The form could not be read. Images must be 5 MB or smaller."))
		return
	}

	pkg := models.Package{}
	if isEdit {
		idStr := r.URL.Query().Get("id")
		if idStr == "" {
			idStr = r.FormValue("id")
		}
		id, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil || id == 0 {
			packageFail(w, r, packageInputError("Invalid package ID."))
			return
		}
		if err := uadmin.Get(&pkg, "id = ?", id); err != nil {
			packageFail(w, r, packageInputError("Package not found."))
			return
		}
	}

	if err := fillPackage(r, &pkg); err != nil {
		packageFail(w, r, err)
		return
	}

	oldImage := pkg.Image
	newImage, err := uploadPackageImage(r)
	if err != nil {
		packageFail(w, r, err)
		return
	}
	if newImage != "" {
		pkg.Image = newImage
	}

	pkg.Save()
	deletePackageImage(newImage)
	packageFail(w, r, err)

	if newImage != "" {
		deletePackageImage(oldImage)
	}

	uadmin.ReturnJSON(w, r, map[string]interface{}{
		"status":     "ok",
		"package_id": pkg.ID,
	})
}

func fillPackage(r *http.Request, pkg *models.Package) error {
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		return packageInputError("Package name is required.")
	}

	classes, err := formInt(r, "numberOfClasses", 1, "Number of classes must be a whole number of 1 or more.")
	if err != nil {
		return err
	}
	freeClasses, err := formInt(r, "numberOfFreeClasses", 0, "Free classes must be a whole number of 0 or more.")
	if err != nil {
		return err
	}
	duration, err := formInt(r, "classDurationInMinutes", 1, "Class duration must be at least 1 minute.")
	if err != nil {
		return err
	}

	price, err := strconv.ParseFloat(strings.TrimSpace(r.FormValue("price")), 64)
	if err != nil || math.IsNaN(price) || math.IsInf(price, 0) || price < 0 {
		return packageInputError("Price must be 0 or more.")
	}
	price = math.Round(price*100) / 100

	from, err := time.ParseInLocation(packageDateLayout, strings.TrimSpace(r.FormValue("validFrom")), time.Local)
	if err != nil {
		return packageInputError("Valid From must be a valid date.")
	}
	until, err := time.ParseInLocation(packageDateLayout, strings.TrimSpace(r.FormValue("validUntil")), time.Local)
	if err != nil {
		return packageInputError("Valid Until must be a valid date.")
	}
	if until.Before(from) {
		return packageInputError("Valid Until can't be earlier than Valid From.")
	}

	y, m, d := until.Date()
	until = time.Date(y, m, d, 23, 59, 59, 0, time.Local)

	active, _ := strconv.ParseBool(r.FormValue("active"))

	pkg.Name = name
	pkg.NumberOfClasses = classes
	pkg.NumberOfFreeClasses = freeClasses
	pkg.ClassDurationInMinutes = duration
	pkg.Price = price
	pkg.ValidFrom = &from
	pkg.ValidUntil = &until
	pkg.Active = active

	return nil
}

func formInt(r *http.Request, field string, lowest int, message string) (int, error) {
	v, err := strconv.Atoi(strings.TrimSpace(r.FormValue(field)))
	if err != nil || v < lowest {
		return 0, packageInputError(message)
	}
	return v, nil
}

// uploadPackageImage checks the uploaded "image" file (if any), sends it to the
// bucket and returns the stored path to keep in Package.Image. It returns ""
// when no file was sent.
func uploadPackageImage(r *http.Request) (string, error) {
	file, header, err := r.FormFile("image")
	if errors.Is(err, http.ErrMissingFile) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	defer file.Close()

	if header.Size > maxPackageImageSize {
		return "", packageInputError("Image must be 5 MB or smaller.")
	}

	head := make([]byte, 512)
	n, err := io.ReadFull(file, head)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return "", err
	}
	ext, ok := packageImageTypes[http.DetectContentType(head[:n])]
	if !ok {
		return "", packageInputError("Image must be a JPG, PNG, GIF or WebP file.")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	random := make([]byte, 8)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}

	return utils.UploadToFilebase(file, "package-"+hex.EncodeToString(random)+ext)
}

func deletePackageImage(stored string) {
	if stored == "" {
		return
	}
	if err := utils.DeleteFromFilebase(filepath.Base(stored)); err != nil {
		uadmin.Trail(uadmin.ERROR, "Failed to delete package image from Filebase: %v", err)
	}
}

func packageFail(w http.ResponseWriter, r *http.Request, err error) {
	message := "Something went wrong while saving the package."

	var inputErr packageInputError
	if errors.As(err, &inputErr) {
		message = inputErr.Error()
	} else {
		uadmin.Trail(uadmin.ERROR, "PackageHandler: %v", err)
	}

	uadmin.ReturnJSON(w, r, map[string]interface{}{
		"status":  "error",
		"message": message,
	})
}
