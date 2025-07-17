package orm

import (
	"database/sql/driver"
	"testing"
)

func TestJSONData_Marshal(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		wantErr bool
	}{
		{
			name: "marshal struct",
			input: struct {
				Name string `json:"name"`
				Age  int    `json:"age"`
			}{Name: "John", Age: 30},
			wantErr: false,
		},
		{
			name:    "marshal nil",
			input:   nil,
			wantErr: false,
		},
		{
			name:    "marshal map",
			input:   map[string]interface{}{"key": "value"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := &JSONData{}
			err := j.Marshal(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Marshal() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestJSONData_Unmarshal(t *testing.T) {
	type testStruct struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	j := &JSONData{}
	err := j.Marshal(testStruct{Name: "John", Age: 30})
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var result testStruct
	err = j.Unmarshal(&result)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if result.Name != "John" || result.Age != 30 {
		t.Errorf("Unmarshal() got = %+v, want Name=John Age=30", result)
	}
}

func TestJSONData_Scan(t *testing.T) {
	tests := []struct {
		name    string
		value   interface{}
		wantErr bool
	}{
		{
			name:    "scan bytes",
			value:   []byte(`{"key":"value"}`),
			wantErr: false,
		},
		{
			name:    "scan string",
			value:   `{"key":"value"}`,
			wantErr: false,
		},
		{
			name:    "scan nil",
			value:   nil,
			wantErr: false,
		},
		{
			name:    "scan unsupported type",
			value:   123,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := &JSONData{}
			err := j.Scan(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("Scan() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestJSONData_Value(t *testing.T) {
	tests := []struct {
		name    string
		data    JSONData
		want    driver.Value
		wantErr bool
	}{
		{
			name:    "value with data",
			data:    JSONData{RawMessage: []byte(`{"key":"value"}`)},
			want:    []byte(`{"key":"value"}`),
			wantErr: false,
		},
		{
			name:    "value with empty data",
			data:    JSONData{},
			want:    nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.data.Value()
			if (err != nil) != tt.wantErr {
				t.Errorf("Value() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got == nil && tt.want == nil {
				return
			}
			if got != nil && tt.want != nil {
				gotBytes := got.([]byte)
				wantBytes := tt.want.([]byte)
				if string(gotBytes) != string(wantBytes) {
					t.Errorf("Value() = %v, want %v", got, tt.want)
				}
			} else if got != tt.want {
				t.Errorf("Value() = %v, want %v", got, tt.want)
			}
		})
	}
}
