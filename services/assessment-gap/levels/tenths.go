package levels

import (
	"encoding/json"
	"fmt"
	"math"
)

// Tenths is a level or percentage stored as an exact tenth.
// 28 is 2.8 and 12 is 1.2.
type Tenths int

func (t Tenths) String() string {
	value := int(t)
	sign := ""
	if value < 0 {
		sign = "-"
		value = -value
	}
	return fmt.Sprintf("%s%d.%d", sign, value/10, value%10)
}

func (t Tenths) MarshalJSON() ([]byte, error) {
	return []byte(t.String()), nil
}

func (t *Tenths) UnmarshalJSON(data []byte) error {
	var value float64
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	parsed, err := TenthsFromFloat(value, 1000)
	if err != nil {
		return err
	}
	*t = parsed
	return nil
}

// TenthsFromFloat accepts one decimal place. maxTenths bounds a level (40)
// or a percentage (1000).
func TenthsFromFloat(value float64, maxTenths int) (Tenths, error) {
	scaled := value * 10
	tenths := int(math.Round(scaled))
	if math.Abs(scaled-float64(tenths)) > 0.001 {
		return 0, ErrInvalid
	}
	if tenths < 0 || tenths > maxTenths {
		return 0, ErrInvalid
	}
	return Tenths(tenths), nil
}
