package api

import (
	"encoding/json"
	"errors"
)

// UnmarshalJSON reads the heterogeneous pair the predictions endpoint uses for
// a ranking prediction: ["frc177", [1, 1, 2.5, 0, 0]].
//
// The numbers after the predicted rank are not documented and have changed
// shape between seasons, so anything in the list that is not a number is
// dropped rather than failing the whole response: a prediction is a convenience
// and a season TBA models differently should still print its ranks.
func (r *RankingPrediction) UnmarshalJSON(b []byte) error {
	var pair []json.RawMessage
	if err := json.Unmarshal(b, &pair); err != nil {
		return err
	}
	if len(pair) == 0 {
		return errors.New("ranking prediction is an empty array")
	}
	if err := json.Unmarshal(pair[0], &r.TeamKey); err != nil {
		return err
	}
	r.Values = nil
	if len(pair) < 2 {
		return nil
	}
	var values []interface{}
	if err := json.Unmarshal(pair[1], &values); err != nil {
		return nil
	}
	for _, v := range values {
		if f, ok := v.(float64); ok {
			r.Values = append(r.Values, f)
		}
	}
	return nil
}

// MarshalJSON writes the pair shape back, so that a round trip through the
// struct does not silently change the document.
func (r RankingPrediction) MarshalJSON() ([]byte, error) {
	values := r.Values
	if values == nil {
		values = []float64{}
	}
	return json.Marshal([]interface{}{r.TeamKey, values})
}
