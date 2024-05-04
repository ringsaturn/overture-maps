package overturemaps

type Propertyable interface {
	ToProperties() any
}

type SourceItem struct {
	Property string `parquet:"property,plain"`
}

type OvertureCommon struct {
	// Top-level Overture theme this feature belongs to
	//
	// Possible values:
	// - "admins"
	// - "base"
	// - "buildings"
	// - "divisions"
	// - "places"
	// - "transportation"
	Theme string `parquet:"theme,plain"`

	// Specific feature type within the theme.
	//
	// Possible values:
	// - "administrative_boundary"
	// - "boundary"
	// - "building"
	// - "connector"
	// - "division"
	// - "division_area"
	// - "infrastructure"
	// - "land"
	// - "land_use"
	// - "locality"
	// - "locality_area"
	// - "building_part"
	// - "place"
	// - "segment"
	// - "water"
	Type string `parquet:"type,plain"`

	// Version number of the feature, incremented in each Overture release where
	// the geometry or attributes of this feature changed.
	//
	// Possible values : >= 0
	Version int

	// Timestamp when the feature was last updated
	UpdateTime string `parquet:"update_time,plain"`

	// The array of source information for the properties of a given feature,
	// with each entry being a source object which lists the property in JSON
	// Pointer notation and the dataset that specific value came from.
	// All features must have a root level source which is the default source if
	// a specific property's source is not specified.
	Sources []SourceItem `parquet:"sources,struct"`
}

type BBox struct {
	XMin float32 `parquet:"xmin"`
	YMin float32 `parquet:"ymin"`
	XMax float32 `parquet:"xmax"`
	YMax float32 `parquet:"ymax"`
}
