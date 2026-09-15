package views

import (
	"net/http"

	"github.com/uadmin/uadmin"
)

func Render(w http.ResponseWriter, r *http.Request, tpl string, context map[string]interface{}) {
	templateList := []string{}
	templateList = append(templateList, "./templates/admin/base.html")

	path := "./templates/admin/" + tpl + ".html"
	templateList = append(templateList, path)
	uadmin.RenderMultiHTML(w, r, templateList, context)
}
