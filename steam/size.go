package steam

import (
	"encoding/json"
	"errors"
)

type ItemSize uint64

func (u *ItemSize) UnmarshalJSON(bs []byte) error {
	var i uint64
	if err := json.Unmarshal(bs, &i); err == nil {
		*u = ItemSize(i)
		return nil
	}

	var s string
	if err := json.Unmarshal(bs, &s); err != nil {
		return errors.New("expected a string or an integer")
	}

	if err := json.Unmarshal([]byte(s), &i); err != nil {
		return err
	}

	*u = ItemSize(i)
	return nil
}
