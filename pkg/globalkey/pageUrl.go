package globalkey

import "fmt"

const UrlNotFound = "/err/404"
const UrlLogin = "/login"

// 访问document的路径
const urlDocument = "/documents/%s/edit"

func UrlDocument(documentId string) string {
	return fmt.Sprintf(urlDocument, documentId)
}
