package ringbuffer

import (
	"reflect"
	"testing"
)

func Test_ringBuffer_Push(t *testing.T) {
	tests := []struct {
		name   string
		ring   *RingBuffer[string]
		values []string
		want   []interface{}
	}{
		{
			name: "push one value to empty ring",
			ring: New[string](3, 1),
			values: []string{
				"a",
			},
			want: []interface{}{
				"a",
				nil,
				nil,
			},
		},
		{
			name: "push four values to empty ring of capacity 3",
			ring: New[string](3, 1),
			values: []string{
				"a",
				"b",
				"c",
				"d",
			},
			want: []interface{}{
				"d",
				"b",
				"c",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pointerValueSlice := make([]*string, len(tt.values))
			for i, v := range tt.values {
				s := v
				pointerValueSlice[i] = &s
			}
			pointerWantSlice := make([]*string, len(tt.want))
			for i, v := range tt.want {
				if v == nil {
					pointerWantSlice[i] = nil
				} else {
					s := v.(string)
					pointerWantSlice[i] = &s
				}
			}
			tt.ring.PushAndWait(pointerValueSlice...)
			if !reflect.DeepEqual(tt.ring.data, pointerWantSlice) {
				got := make([]string, len(tt.ring.data))
				for i, v := range tt.ring.data {
					if v == nil {
						got[i] = ""
					} else {
						got[i] = *v
					}
				}
				t.Errorf("ringBuffer.Push() = %v, want %v", got, tt.want)
			}
			tt.ring.Close()
		})
	}
}
