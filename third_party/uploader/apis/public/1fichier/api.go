package fichier

import "uploader/apis"

var Backend = new(fichier)

type fichier struct {
	apis.Backend
	pwd    string
	apiKey string
	email  string
	useFTP bool
	resp   string
	remove string
}

func (b *fichier) SetPassword(v string) { b.pwd = v }
func (b *fichier) SetAPIKey(v string)   { b.apiKey = v }
func (b *fichier) SetEmail(v string)    { b.email = v }
func (b *fichier) SetFTP(v bool)        { b.useFTP = v }
