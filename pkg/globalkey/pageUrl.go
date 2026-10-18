package globalkey

import "fmt"

const UrlNotFound = "/err/404"
const UrlLogin = "/login"

// 访问document的路径
const urlDocument = "/documents/%s/edit"
const urlPublicDocument = "/public_documents/%s"
const urlGrant = "/grant/%s"

func UrlDocument(documentId string) string {
	return fmt.Sprintf(urlDocument, documentId)
}

func UrlPublicDocument(snapshotId string) string {
	return fmt.Sprintf(urlPublicDocument, snapshotId)
}

func UrlGrant(grantId string) string {
	return fmt.Sprintf(urlGrant, grantId)
}
