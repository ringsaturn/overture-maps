package overturemaps

type LocalityAreaRow struct {
	OvertureCommon

	// A feature ID.
	//
	// This may be an ID associated with the Global Entity Reference System (GERS)
	// if—and-only-if the feature represents an entity that is part of GERS.
	ID string `parquet:"id,plain"`

	// No extra propertie(s) are authorized in this object
	Geometry []byte `parquet:"geometry,plain"`

	BBox *BBox `parquet:"bbox,struct"`

	// References specific feature of locality type
	LocalityID *string `parquet:"locality_id,plain"`
}

type LocalityAreaRowProperties struct {
	ID         string         `json:"id"`
	BBox       BBox           `json:"bbox"`
	LocalityID string         `json:"locality_id"`
	Base       OvertureCommon `json:"base"`
}

func (row *LocalityAreaRow) ToProperties() *LocalityAreaRowProperties {
	p := &LocalityAreaRowProperties{
		ID:         row.ID,
		LocalityID: *row.LocalityID,
		Base:       row.OvertureCommon,
	}
	if row.BBox != nil {
		p.BBox = *row.BBox
	}
	return p
}
