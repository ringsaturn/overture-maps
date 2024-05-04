package overturemaps

type LocalityNameRule struct {
	Variant  string  `parquet:"variant,plain"`
	Value    string  `parquet:"value,plain"`
	Language *string `parquet:"language,plain"`
}

type LocalityName struct {
	Primary string             `parquet:"primary,plain"`
	Common  map[string]string  `parquet:"common"`
	Rules   []LocalityNameRule `parquet:"rules,struct"`
}

type LocalityRow struct {
	OvertureCommon

	// A feature ID.
	//
	// This may be an ID associated with the Global Entity Reference System (GERS)
	// if—and-only-if the feature represents an entity that is part of GERS.
	ID string `parquet:"id,plain"`

	// No extra propertie(s) are authorized in this object
	Geometry []byte `parquet:"geometry,plain"`

	BBox *BBox `parquet:"bbox,struct"`

	// Flag that specifies if feature is maritime
	// (i.e., a boundary at a particular distance from a jurisdiction's coastline)
	IsMaritime *bool `parquet:"is_maritime,boolean"`

	// Hierarchical level for administrative entity or border.
	// E.g. in United States, Country locality representing United States has
	// adminLevel=1, States have adminLevel=2, Counties have adminLevel=3.
	//
	// Possible values: >= 0 ADN <= 6
	AdminLevel *int `parquet:"admin_level,int64"`

	// Optional value that indicates if the boundary needs special rendering logic.
	//
	// Possible values : "disputed", "hidden", "visible"
	GeopolDisplay *string `parquet:"geopol_display,plain"`

	// populated area types.
	//
	// Possible values: "administrative_locality", "named_locality"
	Subtype *string `parquet:"subtype,plain"`

	// Describes the entity's type in the categorical nomenclature used locally.
	//
	// Possible values:
	// - "country"
	// - "county"
	// - "state"
	// - "region"
	// - "province"
	// - "district"
	// - "city"
	// - "town"
	// - "village"
	// - "hamlet"
	// - "borough"
	// - "suburb"
	// - "neighborhood"
	// - "municipality"
	LocalityType *string `parquet:"locality_type,plain"`

	// A wikidata ID if available, as found on https://www.wikidata.org/.
	Wikidata *string `parquet:"wikidata,plain"`

	// Context entity is the most granular entity that logically contains given
	// entity (but doesn't have to contain it spatially due to minor
	// discrepancies in geometries)
	ContextID *string `parquet:"context_id,plain"`

	// Population in the locality.
	Population *int `parquet:"population,int64"`

	ISOCountryCodeAlpha2 *string      `parquet:"iso_country_code_alpha_2,plain"`
	ISOSubCountryCode    *string      `parquet:"iso_sub_country_code,plain"`
	DefaultLanguage      *string      `parquet:"default_language,plain"`
	DrivingSide          *string      `parquet:"driving_side,plain"`
	Names                LocalityName `parquet:"names,struct"`
}

type LocalityRowProperties struct {
	ID                   string         `json:"id"`
	BBox                 BBox           `json:"bbox"`
	AdminLevel           int            `json:"admin_level"`
	GeopolDisplay        string         `json:"geopol_display"`
	Subtype              string         `json:"subtype"`
	LocalityType         string         `json:"locality_type"`
	Wikidata             string         `json:"wikidata"`
	ContextID            string         `json:"context_id"`
	Population           int            `json:"population"`
	ISOCountryCodeAlpha2 string         `json:"iso_country_code_alpha_2"`
	ISOSubCountryCode    string         `json:"iso_sub_country_code"`
	DefaultLanguage      string         `json:"default_language"`
	DrivingSide          string         `json:"driving_side"`
	Names                LocalityName   `json:"names"`
	Base                 OvertureCommon `json:"base"`
}

func (row *LocalityRow) ToProperties() *LocalityRowProperties {
	p := &LocalityRowProperties{}
	p.ID = row.ID
	if row.BBox != nil {
		p.BBox = *row.BBox
	}
	if row.AdminLevel != nil {
		p.AdminLevel = *row.AdminLevel
	}
	if row.GeopolDisplay != nil {
		p.GeopolDisplay = *row.GeopolDisplay
	}
	if row.Subtype != nil {
		p.Subtype = *row.Subtype
	}
	if row.LocalityType != nil {
		p.LocalityType = *row.LocalityType
	}
	if row.Wikidata != nil {
		p.Wikidata = *row.Wikidata
	}
	if row.ContextID != nil {
		p.ContextID = *row.ContextID
	}
	if row.Population != nil {
		p.Population = *row.Population
	}
	if row.ISOCountryCodeAlpha2 != nil {
		p.ISOCountryCodeAlpha2 = *row.ISOCountryCodeAlpha2
	}
	if row.ISOSubCountryCode != nil {
		p.ISOSubCountryCode = *row.ISOSubCountryCode
	}
	if row.DefaultLanguage != nil {
		p.DefaultLanguage = *row.DefaultLanguage
	}
	if row.DrivingSide != nil {
		p.DrivingSide = *row.DrivingSide
	}

	p.Names = row.Names
	p.Base = row.OvertureCommon

	return p
}
