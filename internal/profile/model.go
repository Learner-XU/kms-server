package profile

import "time"

// Profile represents a user's resume/profile.
// Stored as a single JSON blob in MySQL for simplicity.
type Profile struct {
	Username    string          `json:"username"`
	Name        string          `json:"name"`
	Title       string          `json:"title"`
	Bio         string          `json:"bio"`
	Avatar      string          `json:"avatar"`
	Location    string          `json:"location"`
	Email       string          `json:"email"`
	Phone       string          `json:"phone"`
	Website     string          `json:"website"`
	Github      string          `json:"github"`
	Linkedin    string          `json:"linkedin"`
	Twitter     string          `json:"twitter"`
	Skills      []SkillCategory `json:"skills"`
	Experience  []Experience    `json:"experience"`
	Projects    []Project       `json:"projects"`
	Education   []Education     `json:"education"`
	Certificates []string       `json:"certificates"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type SkillCategory struct {
	Category string      `json:"category"`
	Items    []SkillItem `json:"items"`
}

type SkillItem struct {
	Name  string `json:"name"`
	Level int    `json:"level"`
}

type Experience struct {
	Company    string `json:"company"`
	Role       string `json:"role"`
	Period     string `json:"period"`
	Type       string `json:"type"`
	Duration   string `json:"duration"`
	Description string `json:"description"`
}

type Project struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Period      string   `json:"period"`
	Description string   `json:"description"`
	Tech        []string `json:"tech"`
	Icon        string   `json:"icon"`
}

type Education struct {
	School string `json:"school"`
	Degree string `json:"degree"`
	Period string `json:"period"`
}
