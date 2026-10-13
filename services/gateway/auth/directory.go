package auth

import "errors"

const (
	OrganisationID = "8f0c2d44-6a1e-4b7a-9c3d-1e5f7a9b0c21"
	AdaPersonID    = "3c1e9a70-5b24-4d8f-a6e1-2f0b8c4d6e11"
	EliPersonID    = "7d2b6f18-9c40-4e15-8a73-5b1d0e6f4a22"
	MayaPersonID   = "1a9e4c55-2d67-4f08-b3e6-8c0a7d5f1b33"
)

var ErrUnknownPerson = errors.New("unknown person")

// Person is a fixed development identity. ManagerID is the person's manager.
// ManagerOf lists the people this person manages.
type Person struct {
	Key            string
	Name           string
	PersonID       string
	OrganisationID string
	Roles          []string
	ManagerID      string
	ManagerOf      []string
}

func personByID(personID string) (Person, bool) {
	for _, person := range developmentPeople() {
		if person.PersonID == personID {
			return person, true
		}
	}
	return Person{}, false
}

func developmentPeople() map[string]Person {
	return map[string]Person{
		"ada": {
			Key:            "ada",
			Name:           "Ada Admin",
			PersonID:       AdaPersonID,
			OrganisationID: OrganisationID,
			Roles:          []string{RoleAdmin},
			ManagerOf:      []string{},
		},
		"eli": {
			Key:            "eli",
			Name:           "Eli Employee",
			PersonID:       EliPersonID,
			OrganisationID: OrganisationID,
			Roles:          []string{RoleEmployee},
			ManagerID:      MayaPersonID,
			ManagerOf:      []string{},
		},
		"maya": {
			Key:            "maya",
			Name:           "Maya Manager",
			PersonID:       MayaPersonID,
			OrganisationID: OrganisationID,
			Roles:          []string{RoleManager},
			ManagerOf:      []string{EliPersonID},
		},
	}
}
