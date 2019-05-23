package domain

// NamedEntity represents a named-entity as it relates to a document.
type NamedEntity struct {
	Frequency int    `json:"frequency"` // Number of occurrences in document.
	Entity    string `json:"entity"`    // String content value of entity.
	Stemmed   string `json:"stemmed"`   // Stemmed form of named entity.
	Label     string `json:"label"`     // Category of named entity.
	POS       string `json:"pos"`       // Part-of-speech.
}

type NamedEntities []NamedEntity
