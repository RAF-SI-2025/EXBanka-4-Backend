package models

type Company struct {
	ID                 int64
	Name               string
	RegistrationNumber string
	PIB                string
	ActivityCode       string
	Address            string
	OwnerClientID      int64
}
