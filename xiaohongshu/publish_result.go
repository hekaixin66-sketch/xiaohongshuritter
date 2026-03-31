package xiaohongshu

import "strings"

type ProductBindReport struct {
	Status            string
	Count             int
	ProductsRequested []string
	ProductsResolved  []string
	ProductsMissing   []string
	VerifyConfidence  float64
}

type ProductBindError struct {
	Report ProductBindReport
	Err    error
}

func (e *ProductBindError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err == nil {
		return "product bind failed"
	}
	return e.Err.Error()
}

func (e *ProductBindError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

type PublishArtifacts struct {
	ProductBind ProductBindReport
}

func newProductBindReport(products []string) ProductBindReport {
	report := ProductBindReport{
		ProductsRequested: copyStringSlice(products),
	}
	if len(products) == 0 {
		report.Status = "skipped"
	}
	return report
}

func copyStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, len(values))
	copy(out, values)
	return out
}

func (r *ProductBindReport) finalize() {
	if r == nil {
		return
	}
	r.Count = len(r.ProductsResolved)
	switch {
	case len(r.ProductsRequested) == 0:
		r.Status = "skipped"
		r.VerifyConfidence = 1
	case len(r.ProductsMissing) == 0:
		r.Status = "success"
		r.VerifyConfidence = 1
	case len(r.ProductsResolved) == 0:
		r.Status = "failed"
		r.VerifyConfidence = 0
	default:
		r.Status = "partial"
		r.VerifyConfidence = 0.5
	}
}

func appendUnique(values []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
