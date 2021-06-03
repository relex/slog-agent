package base

// FieldSetExtractor extracts values from log records by pre-defined set of fields
type FieldSetExtractor struct {
	locators       []LogFieldLocator
	fieldSetBuffer []string
}

// NewFieldSetExtractor creates a FieldSetExtractor
func NewFieldSetExtractor(locators []LogFieldLocator) *FieldSetExtractor {
	return &FieldSetExtractor{
		locators:       locators,
		fieldSetBuffer: make([]string, len(locators)),
	}
}

// Extract extracts field set from the given log record and returns transient field values
// Returned slices and values are only usable until next call. They MUST be copied for storing.
func (ex *FieldSetExtractor) Extract(record *LogRecord) []string {
	transientFieldSet := ex.fieldSetBuffer
	for i, loc := range ex.locators {
		transientFieldSet[i] = loc.Get(record.Fields)
	}
	return transientFieldSet
}
