// Vikunja is a to-do list application to facilitate your life.
// Copyright 2018-present Vikunja and contributors. All rights reserved.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package models

import (
	"math"
	"testing"
)

func TestCalculateWeightedProgress(t *testing.T) {
	tests := []struct {
		name  string
		items []progressItem
		want  float64
	}{
		{
			name:  "no items",
			items: nil,
			want:  0,
		},
		{
			name: "single unweighted item not done",
			items: []progressItem{
				{Weight: 0, Done: false},
			},
			want: 0,
		},
		{
			name: "single unweighted item done",
			items: []progressItem{
				{Weight: 0, Done: true},
			},
			want: 1,
		},
		{
			name: "three unweighted items, one done",
			items: []progressItem{
				{Weight: 0, Done: true},
				{Weight: 0, Done: false},
				{Weight: 0, Done: false},
			},
			// 33% rounded
			want: 0.33,
		},
		{
			name: "weighted + unweighted mix, weighted one done",
			// weighted 10 + two unweighted (sharing 90, i.e. 45 each)
			// only the weighted one is done → 10%
			items: []progressItem{
				{Weight: 10, Done: true},
				{Weight: 0, Done: false},
				{Weight: 0, Done: false},
			},
			want: 0.10,
		},
		{
			name: "weighted + unweighted mix, one unweighted done",
			// weighted 10 + two unweighted (sharing 90, i.e. 45 each)
			// one unweighted is done → 45% (rounded)
			items: []progressItem{
				{Weight: 10, Done: false},
				{Weight: 0, Done: true},
				{Weight: 0, Done: false},
			},
			want: 0.45,
		},
		{
			name: "weighted + unweighted mix, weighted + one unweighted done",
			items: []progressItem{
				{Weight: 10, Done: true},
				{Weight: 0, Done: true},
				{Weight: 0, Done: false},
			},
			// 10 + 45 = 55
			want: 0.55,
		},
		{
			name: "all weighted and summing to 100",
			items: []progressItem{
				{Weight: 25, Done: true},
				{Weight: 25, Done: true},
				{Weight: 50, Done: false},
			},
			want: 0.50,
		},
		{
			name: "weights overflow 100 → equal split fallback",
			items: []progressItem{
				{Weight: 60, Done: true},
				{Weight: 60, Done: false},
				{Weight: 0, Done: true},
			},
			// overflow → 2 of 3 done = 67%
			want: 0.67,
		},
		{
			name: "unweighted half done, two items",
			items: []progressItem{
				{Weight: 0, Done: true},
				{Weight: 0, Done: false},
			},
			want: 0.50,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := calculateWeightedProgress(tc.items)
			if math.Abs(got-tc.want) > 0.0001 {
				t.Errorf("calculateWeightedProgress() = %v, want %v", got, tc.want)
			}
		})
	}
}
