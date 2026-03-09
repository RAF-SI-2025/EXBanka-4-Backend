package models

type Employee struct {
	ID            int64
	Ime           string
	Prezime       string
	DatumRodjenja string
	Pol           string
	Email         string
	BrojTelefona  string
	Adresa        string
	Username      string
	Pozicija      string
	Departman     string
	Aktivan       bool
	Dozvole       []string
}
