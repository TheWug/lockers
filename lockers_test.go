package lockers

import (
	"testing"
	"errors"
	"fmt"
)

func Test_SizeSpec_Contains(t *testing.T) {
	type X struct {
		first, second SizeSpec
		forward, reverse bool
	}
	tests := map[string]X{
		"self": X{SizeSpec{10, 10, 10}, SizeSpec{10, 10, 10}, true, true},
		"bigger-x": X{SizeSpec{10, 10, 10}, SizeSpec{11, 10, 10}, false, true},
		"bigger-y": X{SizeSpec{10, 10, 10}, SizeSpec{10, 11, 10}, false, true},
		"bigger-z": X{SizeSpec{10, 10, 10}, SizeSpec{10, 10, 11}, false, true},
		"skewed": X{SizeSpec{10, 10, 10}, SizeSpec{9,  11, 10}, false, false},
		"denormalized": X{SizeSpec{10, 11, 12}, SizeSpec{12, 11, 10}, false, false},
	}
	
	for k, v := range tests {
		t.Run(k, func(t *testing.T) {
			if v.first.Contains(v.second) != v.forward {
				t.Errorf("containment failure: %v CONTAINS %v (%t, expected %t)", v.first, v.second, !v.forward, v.forward)
			}
			if v.second.Contains(v.first) != v.reverse {
				t.Errorf("containment failure: %v CONTAINS %v (%t, expected %t)", v.second, v.first, !v.reverse, v.reverse)
			}
		})
	}
}

func Test_SizeSpec_Volume(t *testing.T) {
	type X struct {
		value SizeSpec
		answer int64
	}
	
	tests := map[string]X{
		"10x10x10": X{SizeSpec{10,10,10}, 10*10*10},
		"10x10x11": X{SizeSpec{10,10,11}, 10*10*11},
		"10x12x11": X{SizeSpec{10,12,11}, 10*12*11},
		"13x12x11": X{SizeSpec{13,12,11}, 13*12*11},
		"negative": X{SizeSpec{-1,1,1}, -1},
	}
	
	for k, v := range tests {
		t.Run(k, func(t *testing.T) {
			if v.value.Volume() != v.answer {
				t.Errorf("VOLUME %v (expected %d, got %d)", v.value, v.answer, v.value.Volume())
			}
		})
	}
}

func Test_SizeSpec_Normalize(t *testing.T) {
	tests := map[string]SizeSpec{
		"rearrange-1": SizeSpec{Length: 1, Width: 3, Height: 5},
		"rearrange-2": SizeSpec{Length: 1, Width: 5, Height: 3},
		"rearrange-3": SizeSpec{Length: 3, Width: 1, Height: 5},
		"rearrange-4": SizeSpec{Length: 3, Width: 5, Height: 1},
		"rearrange-5": SizeSpec{Length: 5, Width: 3, Height: 1},
		"rearrange-6": SizeSpec{Length: 5, Width: 1, Height: 3},
		"negate-1": SizeSpec{Length: -5, Width: 3, Height: 1},
		"negate-2": SizeSpec{Length: 5, Width: -3, Height: 1},
		"negate-3": SizeSpec{Length: 5, Width: 3, Height: -1},
	}
	
	answer := SizeSpec{Length: 5, Width: 3, Height: 1}
	
	for k, v := range tests {
		t.Run(k, func(t *testing.T) {
			if v.Normalize() != answer {
				t.Errorf("%v mis-normalized to %v", v, v.Normalize())
			}
		})
	}
}

type MockInventory struct {
	CompareFrom, CompareTo *LockerControlSpec
}

func (mcs MockInventory) ControlSpec(size_id LockerSize) *LockerControlSpec {
	if size_id == LockerSize(0) {
		return mcs.CompareFrom
	}
	
	return mcs.CompareTo
}

func Test_LockerSize_Before(t *testing.T) {
	type X struct {
		cmp LockerControlSpec
		expected bool
	}
	
	inv := MockInventory{
		CompareFrom: &LockerControlSpec{
			VirtualCapacity: 50,
			Size: SizeSpec{5, 5, 5},
		},
	}
	
	tests := map[string]X{
		"75-6": X{LockerControlSpec{VirtualCapacity: 75, Size: SizeSpec{6,6,6}}, false},
		"75-5": X{LockerControlSpec{VirtualCapacity: 75, Size: SizeSpec{5,5,5}}, false},
		"75-4": X{LockerControlSpec{VirtualCapacity: 75, Size: SizeSpec{4,4,4}}, false},
		"50-6": X{LockerControlSpec{VirtualCapacity: 50, Size: SizeSpec{6,6,6}}, true},
		"50-5": X{LockerControlSpec{VirtualCapacity: 50, Size: SizeSpec{5,5,5}}, false},
		"50-4": X{LockerControlSpec{VirtualCapacity: 50, Size: SizeSpec{4,4,4}}, false},
		"25-6": X{LockerControlSpec{VirtualCapacity: 25, Size: SizeSpec{6,6,6}}, true},
		"25-5": X{LockerControlSpec{VirtualCapacity: 25, Size: SizeSpec{5,5,5}}, true},
		"25-4": X{LockerControlSpec{VirtualCapacity: 25, Size: SizeSpec{4,4,4}}, true},
	}
	
	for k, v := range tests {
		t.Run(k, func(t *testing.T) {
			inv.CompareTo = &v.cmp
			if LockerSize(0).Before(LockerSize(1), inv) != v.expected {
				t.Errorf("Unexpected BEFORE result: %v %v (%t, should be %t)", inv.CompareFrom, inv.CompareTo, !v.expected, v.expected)
			}
		})
	}
}

func Test_LockerControlSpec_Full(t *testing.T) {
	spec := LockerControlSpec{}
	if !spec.Full() {
		t.Errorf("%v not full, should have been", spec)
	}
	
	spec.Lockers = append(spec.Lockers, 1)
	if spec.Full() {
		t.Errorf("%v full, should not have been", spec)
	}
}

func Test_Inventory_ControlSpec(t *testing.T) {
	inv := Inventory{
		Control: make(map[LockerSize]*LockerControlSpec),
	}
	ctrl := &LockerControlSpec{}
	inv.Control[LockerSize(1)] = ctrl
	if inv.ControlSpec(LockerSize(1)) != ctrl {
		t.Errorf("Failed to retrieve LockerControlSpec")
	}
	if inv.ControlSpec(LockerSize(2)) != nil {
		t.Errorf("Spuriously retrieved LockerControlSpec")
	}
}

func Example_Locker_PutFetch() {
	locker := Locker{
	}
	
	pkg := Package{
	}
	
	no_error := errors.New("No error")
	
	e := locker.Put(&pkg)
	if e == nil { e = no_error }
	fmt.Println(e.Error()) // no error
	
	p, e := locker.Fetch()
	if e == nil { e = no_error }
	fmt.Println(e.Error()) // no error
	if p != &pkg {
		fmt.Println("got a different package back?")
	}
	
	locker = Locker{Contents: &pkg}
	pkg = Package{}
	
	e = locker.Put(&pkg)
	if e == nil { e = no_error }
	fmt.Println(e.Error()) // locker is not empty
	
	locker = Locker{}
	pkg = Package{StoredIn: &locker}
	
	e = locker.Put(&pkg)
	if e == nil { e = no_error }
	fmt.Println(e.Error()) // package already in locker
	
	locker = Locker{}
	pkg = Package{}
	
	_, e = locker.Fetch()
	if e == nil { e = no_error }
	fmt.Println(e.Error()) // Tried to fetch from empty locker
	
	// Output:
	// No error
	// No error
	// Locker is not empty
	// Package already in locker
	// Tried to fetch from empty locker
}

func CompareControls(t *testing.T, a, b *LockerControlSpec, ia, ib *Inventory) bool {
	// compare the size
	if a.Size != b.Size {
		return false
	}
	
	// compare the biggerthan values by comparing the sizes they indirect against
	if len(a.BiggerThan) != len(b.BiggerThan) {
		return false
	}
	sizes := make(map[SizeSpec]bool)
	for _, x := range a.BiggerThan {
		sizes[ia.Control[x].Size] = true
	}
	for _, x := range b.BiggerThan {
		delete(sizes, ib.Control[x].Size)
	}
	if len(sizes) != 0 {
		return false
	}
	
	// compare the smallerthan values the same way
	if len(a.SmallerThan) != len(b.SmallerThan) {
		return false
	}
	sizes = make(map[SizeSpec]bool)
	for _, x := range a.SmallerThan {
		sizes[ia.Control[x].Size] = true
	}
	for _, x := range b.SmallerThan {
		delete(sizes, ib.Control[x].Size)
	}
	if len(sizes) != 0 {
		return false
	}
	
	// check the length of the lockers. the indices themselves may vary but there
	// should be the same number of them
	if len(a.Lockers) != len(b.Lockers) {
		return false
	}
	
	// check the virtual capacity
	if a.VirtualCapacity != b.VirtualCapacity {
		return false
	}
	
	return true
}

func ValidateInventory(t *testing.T, a *Inventory) (bool, string) {
	t.Helper()
	
	if len(a.Sizes) != len(a.Control) {
		return false, "len(sizes) and len(control) mismatch"
	}
	
	for k, v := range a.Control {
		if k != v.SizeId { return false, "inconsistent ControlSpec.SizeId to key" }
		if k != a.Sizes[v.Size] { return false, "inconsistent Sizes[ControlSpec.Size] to key" }
		for _, i := range v.Lockers {
			if i >= len(a.Lockers) { return false, "locker index in ControlSpec.Lockers too big" }
			if a.Lockers[i].SizeId != k { return false, "inconsistent locker SizeId to ControlSpec" }
		}
	}
	
	for k, v := range a.LockersById {
		if v >= len(a.Lockers) { return false, "locker index in LockersById too big" }
		if a.Lockers[v].Id != k { return false, "inconsistent locker id to LockersById" }
	}
	
	return true, ""
}

func CompareInventories(t *testing.T, a, b *Inventory) (bool, string) {
	t.Helper()
	
	// compare Sizes.
	// the keys should be the same, but will be in a randomized order.
	if len(a.Sizes) != len(b.Sizes) {
		return false, ".Sizes lengths not equal"
	}
	
	keys := make(map[SizeSpec]bool)
	
	for k, _ := range a.Sizes {
		keys[k] = true
	}
	
	for k, _ := range b.Sizes {
		delete(keys, k)
	}
	
	if len(keys) != 0 {
		return false, ".Sizes keys/values mismatched"
	}
	
	// compare the control structures. SizeIds may be different between runs, 
	// and thusly they may occur out of order, but we can translate the SizeId
	// using .Sizes
	if len(a.Control) != len(b.Control) {
		return false, ".Control lengths not equal"
	}
	
	for size_id, control := range a.Control {
		remote_size_id, ok := b.Sizes[control.Size]
		if !ok { return false, fmt.Sprintf("missing control size %v", size_id) }
		remote_control, ok := b.Control[remote_size_id]
		if !ok { return false, fmt.Sprintf("missing control for size %v", size_id) }
		
		if !CompareControls(t, control, remote_control, a, b) {
			return false, fmt.Sprintf("Control values not equal:\n%+v\n%+v", control, remote_control)
		}
	}
	
	
	// compare the lockers. they can be out of order, so this is challenging.
	// so instead, create a map of flattened proxy objects and compare those.
	if len(a.Lockers) != len(b.Lockers) {
		return false, ".Lockers lengths not equal"
	}
	
	type PackageProxy struct {
		Size SizeSpec
		Id string
	}
	
	type LockerProxy struct {
		Contents PackageProxy
		Size SizeSpec
	}
	
	locker_proxies := make(map[LockerProxy]int)
	for _, x := range a.Lockers {
		p := LockerProxy{
			Size: a.Control[x.SizeId].Size,
		}
		if x.Contents != nil {
			p.Contents = PackageProxy{
				Id: x.Contents.Id,
				Size: x.Contents.Size,
			}
		}
		
		locker_proxies[p] = locker_proxies[p] + 1
	}
	for _, x := range b.Lockers {
		p := LockerProxy{
			Size: b.Control[x.SizeId].Size,
		}
		if x.Contents != nil {
			p.Contents = PackageProxy{
				Id: x.Contents.Id,
				Size: x.Contents.Size,
			}
		}
		
		locker_proxies[p] = locker_proxies[p] - 1
	}
	for k, v := range locker_proxies {
		if v != 0 {
			return false, fmt.Sprintf("Incorrect count for locker proxy: %+v", k)
		}
	}
	
	// compare lockers by id. the keys are randomly generated, and in general the indices may occur
	// out of order, but since this map has all of the indices and the order doesn't matter, they
	// should at least all be there.
	if len(a.LockersById) != len(b.LockersById) {
		return false, ".LockersById lengths not equal"
	}
	
	locker_indices := make(map[int]bool)
	
	for _, v := range a.LockersById {
		locker_indices[v] = true
	}
	for _, v := range b.LockersById {
		delete(locker_indices, v)
	}
	
	if len(locker_indices) != 0 {
		return false, "leftover locker indices"
	}
	
	return true, ""
}

// Note: this test sometimes spuriously reports less than 100% code coverage in NewInventory.
// The problem is that some code in NewInventory is conditionally executed depending
// on the order of traversal of a map, and go maps are unordered. There is no way to guarantee
// an order such that all code paths are hit.
func Test_New_Inventory(t *testing.T) {
	type X struct {
		size_counts map[SizeSpec]int
		result *Inventory
	}
	
	tests := map[string]X {
		"no-lockers": X{map[SizeSpec]int{}, &Inventory{
			Lockers: make([]Locker, 0),
			Control: make(map[LockerSize]*LockerControlSpec),
			Sizes: make(map[SizeSpec]LockerSize),
			LockersById: make(map[string]int),
			LockersByPackageId: make(map[string]int),
		}},
		"1-type-lockers": X{map[SizeSpec]int{SizeSpec{1,1,1}:3}, &Inventory{
			Lockers: []Locker{
				Locker{"1", 100, nil},
				Locker{"2", 100, nil},
				Locker{"3", 100, nil},
			},
			Control: map[LockerSize]*LockerControlSpec{
				100: &LockerControlSpec{
					SizeId: 100,
					Size: SizeSpec{1,1,1},
					Lockers: []int{0,1,2},
					VirtualCapacity: 3,
				},
			},
			Sizes: map[SizeSpec]LockerSize{
				SizeSpec{1,1,1}: 100,
			},
			LockersById: map[string]int{
				"1": 0,
				"2": 1,
				"3": 2,
			},
			LockersByPackageId: make(map[string]int),
		}},
		"3-type-lockers": X{map[SizeSpec]int{SizeSpec{3,3,3}:2, SizeSpec{1,1,1}:2, SizeSpec{2,2,2}:2}, &Inventory{
			Lockers: []Locker{
				Locker{"1", 100, nil},
				Locker{"2", 100, nil},
				Locker{"3", 200, nil},
				Locker{"4", 200, nil},
				Locker{"5", 300, nil},
				Locker{"6", 300, nil},
			},
			Control: map[LockerSize]*LockerControlSpec{
				100: &LockerControlSpec{
					SizeId: 100,
					Size: SizeSpec{1,1,1},
					SmallerThan: []LockerSize{200, 300},
					Lockers: []int{0,1},
					VirtualCapacity: 6,
				},
				200: &LockerControlSpec{
					SizeId: 200,
					Size: SizeSpec{2,2,2},
					SmallerThan: []LockerSize{300},
					BiggerThan: []LockerSize{100},
					Lockers: []int{2,3},
					VirtualCapacity: 4,
				},
				300: &LockerControlSpec{
					SizeId: 200,
					Size: SizeSpec{3,3,3},
					BiggerThan: []LockerSize{100,200},
					Lockers: []int{4,5},
					VirtualCapacity: 2,
				},
			},
			Sizes: map[SizeSpec]LockerSize{
				SizeSpec{1,1,1}: 100,
				SizeSpec{2,2,2}: 200,
				SizeSpec{3,3,3}: 300,
			},
			LockersById: map[string]int{
				"1": 0,
				"2": 1,
				"3": 2,
				"4": 3,
				"5": 4,
				"6": 5,
			},
			LockersByPackageId: make(map[string]int),
		}},
		"duplicate-lockers": X{map[SizeSpec]int{SizeSpec{2,1,1}:2, SizeSpec{1,2,1}:2}, &Inventory{
			Lockers: []Locker{
				Locker{"1", 100, nil},
				Locker{"2", 100, nil},
				Locker{"3", 100, nil},
				Locker{"4", 100, nil},
			},
			Control: map[LockerSize]*LockerControlSpec{
				100: &LockerControlSpec{
					SizeId: 100,
					Size: SizeSpec{2,1,1},
					Lockers: []int{0,1,2,3},
					VirtualCapacity: 4,
				},
			},
			Sizes: map[SizeSpec]LockerSize{
				SizeSpec{2,1,1}: 100,
			},
			LockersById: map[string]int{
				"1": 0,
				"2": 1,
				"3": 2,
				"4": 3,
			},
			LockersByPackageId: make(map[string]int),
		}},
	}
	
	for k, v := range tests {
		t.Run(k, func(t *testing.T) {
			i := NewInventory(v.size_counts)
			eq, explanation := CompareInventories(t, i, v.result)
			if !eq {
				t.Errorf("Incorrect NewInventory output:\n%+v\nExpected:\n%+v\n%s", i, v.result, explanation)
			}
			valid, explanation := ValidateInventory(t, i)
			if !valid {
				t.Errorf("Invalid or malformed inventory:\n%+v\n%s", i, explanation)
			}
		})
	}
}

func basic(t *testing.T) *Inventory {
	t.Helper()
	
	return &Inventory{
		Lockers: []Locker{
			Locker{"1", 100, nil},
			Locker{"2", 100, nil},
			Locker{"3", 200, nil},
			Locker{"4", 200, nil},
			Locker{"5", 300, nil},
			Locker{"6", 300, nil},
			Locker{"7", 400, nil},
			Locker{"8", 400, nil},
		},
		Control: map[LockerSize]*LockerControlSpec{
			100: &LockerControlSpec{
				SizeId: 100,
				Size: SizeSpec{1,1,1},
				SmallerThan: []LockerSize{200,300,400},
				Lockers: []int{0,1},
				VirtualCapacity: 6,
			},
			200: &LockerControlSpec{
				SizeId: 200,
				Size: SizeSpec{2,2,2},
				SmallerThan: []LockerSize{300,400},
				BiggerThan: []LockerSize{100},
				Lockers: []int{2,3},
				VirtualCapacity: 4,
			},
			300: &LockerControlSpec{
				SizeId: 300,
				Size: SizeSpec{3,3,3},
				SmallerThan: []LockerSize{400},
				BiggerThan: []LockerSize{100,200},
				Lockers: []int{4,5},
				VirtualCapacity: 2,
			},
			400: &LockerControlSpec{
				SizeId: 400,
				Size: SizeSpec{4,4,4},
				BiggerThan: []LockerSize{100,200,300},
				Lockers: []int{},
				VirtualCapacity: 0,
			},
		},
		Sizes: map[SizeSpec]LockerSize{
			SizeSpec{1,1,1}: 100,
			SizeSpec{2,2,2}: 200,
			SizeSpec{3,3,3}: 300,
			SizeSpec{4,4,4}: 400,
		},
		LockersById: map[string]int{
			"1": 0,
			"2": 1,
			"3": 2,
			"4": 3,
			"5": 4,
			"6": 5,
			"7": 6,
			"8": 7,
		},
		LockersByPackageId: make(map[string]int),
	}
}

func cplx(t *testing.T) *Inventory {
	t.Helper()
	
	return &Inventory{
		Lockers: []Locker{
			Locker{"1", 100, nil},
			Locker{"2", 100, nil},
			Locker{"3", 200, nil},
			Locker{"4", 200, nil},
			Locker{"4.5", 200, nil},
			Locker{"5", 300, nil},
			Locker{"6", 300, nil},
			Locker{"7", 400, nil},
			Locker{"8", 400, nil},
		},
		Control: map[LockerSize]*LockerControlSpec{
			100: &LockerControlSpec{
				SizeId: 100,
				Size: SizeSpec{1,1,1},
				SmallerThan: []LockerSize{200,300,400},
				Lockers: []int{0,1},
				VirtualCapacity: 8,
			},
			200: &LockerControlSpec{
				SizeId: 200,
				Size: SizeSpec{5,1,1},
				SmallerThan: []LockerSize{400},
				BiggerThan: []LockerSize{100},
				Lockers: []int{2,3,4},
				VirtualCapacity: 4,
			},
			300: &LockerControlSpec{
				SizeId: 300,
				Size: SizeSpec{3,3,1},
				SmallerThan: []LockerSize{400},
				BiggerThan: []LockerSize{100},
				Lockers: []int{5,6},
				VirtualCapacity: 3,
			},
			400: &LockerControlSpec{
				SizeId: 400,
				Size: SizeSpec{5,5,5},
				BiggerThan: []LockerSize{100,200,300},
				Lockers: []int{7},
				VirtualCapacity: 1,
			},
		},
		Sizes: map[SizeSpec]LockerSize{
			SizeSpec{1,1,1}: 100,
			SizeSpec{5,1,1}: 200,
			SizeSpec{3,3,1}: 300,
			SizeSpec{5,5,5}: 400,
		},
		LockersById: map[string]int{
			"1": 0,
			"2": 1,
			"3": 2,
			"4": 3,
			"4.5": 4,
			"5": 5,
			"6": 6,
			"7": 7,
			"8": 8,
		},
		LockersByPackageId: make(map[string]int),
	}
}

func Test_Inventory_AllocateLocker(t *testing.T) {
	inv := basic(t)
	
	type X struct {
		size SizeSpec
		panics bool
	}
	
	tests := map[string]X{
		"smallest": X{SizeSpec{1,1,1}, false},
		"medium": X{SizeSpec{2,2,2}, false},
		"largest": X{SizeSpec{3,3,3}, false},
		"overallocate": X{SizeSpec{4,4,4}, true},
		"missing": X{SizeSpec{5,5,5}, true},
	}
	
	for k, v := range tests {
		t.Run(k, func(t *testing.T) {
			defer func(){
				r := recover()
				if v.panics && r == nil {
					t.Errorf("Expected panic, but completed normally")
				} else if !v.panics && r != nil {
					t.Errorf("Expected normal completion, but panic'd")
				}
			}()
			
			size := inv.Sizes[v.size]
			var inner_available []int
			var capacity int
			
			if x, ok := inv.Control[size]; ok {
				inner_available = x.Lockers
				capacity = x.VirtualCapacity
			}
			available := make(map[int]bool)
			for _, x := range inner_available {
				available[x] = true
			}
			
			// a should be removed from inv.Control[size].Lockers
			// if the slice has no elements, or if size isn't in Control, it will panic
			a := inv.AllocateLocker(size)
			
			if len(available) != len(inv.Control[size].Lockers) + 1 {
				t.Error("Mismatched lengths between expected and actual lockers free")
			}
			
			delete(available, a)
			for _, x := range inv.Control[size].Lockers {
				delete(available, x)
			}
			
			if len(available) != 0 {
				t.Error("Mismatched values in expected and actual lockers free")
			}
			
			if capacity == inv.Control[size].VirtualCapacity {
				t.Error("Virtual capacity unchanged")
			}
		})
	}
}

func Test_Inventory_AdjustVirtualCapacity(t *testing.T) {
	inv1, inv2, inv3 := basic(t), basic(t), basic(t)
	inv1.Control[LockerSize(100)].VirtualCapacity += 1
	inv2.Control[LockerSize(100)].VirtualCapacity += 2
	inv3.Control[LockerSize(100)].VirtualCapacity += 3
	
	inv2.Control[LockerSize(200)].VirtualCapacity += 2
	inv3.Control[LockerSize(200)].VirtualCapacity += 3
	
	inv3.Control[LockerSize(300)].VirtualCapacity += 3
	
	type X struct {
		add int
		addto LockerSize
		result *Inventory
		panics bool
	}
	
	tests := map[string]X{
		"smallest": X{1, 100, inv1, false},
		"medium": X{2, 200, inv2, false},
		"largest": X{3, 300, inv3, false},
		"missing": X{4, 400, nil, true},
	}
	
	for k, v := range tests {
		t.Run(k, func(t *testing.T) {
			defer func() {
				r := recover()
				if v.panics && r == nil {
					t.Errorf("Expected panic, but completed normally")
				} else if !v.panics && r != nil {
					t.Errorf("Expected normal completion, but panic'd")
				}
			}()
			
			inv := basic(t)
			inv.AdjustVirtualCapacity(v.addto, v.add)
			
			eq, explain := CompareInventories(t, inv, v.result)
			if !eq {
				t.Errorf("Incorrect NewInventory output:\n%+v\nExpected:\n%+v\n%s", inv, v.result, explain)
			}
		})
	}
}

func Test_Inventory_DeallocateLocker(t *testing.T) {
	inv := basic(t)
	
	control, ok := inv.Control[400]
	if !ok || len(control.Lockers) != 0 {
		t.Error("precondition failed")
	}
	
	capacity := control.VirtualCapacity
	
	inv.DeallocateLocker(7)
	inv.DeallocateLocker(6)
	
	available := make(map[int]bool)
	for _, x := range control.Lockers {
		available[x] = true
	}
	
	delete(available, 6)
	delete(available, 7)

	if len(control.Lockers) != 2 || len(available) != 0 {
		t.Error("Element mismatch when checking available list")
	}
	
	if capacity == control.VirtualCapacity {
		t.Errorf("Virtual capacity unchanged: %d %d", capacity, control.VirtualCapacity)
	}
}

func Test_Inventory_GetMostSuitableLockerSize(t *testing.T) {
	inv1, inv2, inv3, inv4 := cplx(t), cplx(t), cplx(t), cplx(t)
	// inv1 is normal and unmodified
	
	// inv2 has all {1,1,1} lockers allocated
	inv2.Control[100].VirtualCapacity -= len(inv2.Control[200].Lockers)
	inv2.Control[100].Lockers = nil
	
	// inv3 has all {1,1,1} lockers allocated, and one {4,1,1} locker resized to {3,3,1}
	inv3.Control[100].VirtualCapacity -= len(inv2.Control[200].Lockers)
	inv3.Control[100].Lockers = nil
	inv3.Control[200].Lockers = []int{2,3}
	inv3.Control[300].Lockers = []int{4,5,6}
	inv3.Lockers[4].SizeId = 300
	inv3.Control[200].VirtualCapacity -= 1
	inv3.Control[300].VirtualCapacity += 1
	
	// inv4 has no available lockers except for small
	for _, c := range inv4.Control {
		if (c.Size == SizeSpec{1,1,1}) {
			c.VirtualCapacity = len(c.Lockers)
			continue
		}
		c.Lockers = nil
		c.VirtualCapacity = 0
	}
	
	type X struct {
		inv *Inventory
		size SizeSpec
		answer SizeSpec
		is_error bool
	}
	
	tests := map[string]X{
		"normal-micro":     X{inv1, SizeSpec{1,0,0}, SizeSpec{1,1,1}, false},
		"normal-small":     X{inv1, SizeSpec{1,1,1}, SizeSpec{1,1,1}, false},
		"normal-med-ambig": X{inv1, SizeSpec{3,1,1}, SizeSpec{5,1,1}, false},
		"normal-med1":      X{inv1, SizeSpec{4,1,1}, SizeSpec{5,1,1}, false},
		"normal-med2":      X{inv1, SizeSpec{2,2,1}, SizeSpec{3,3,1}, false},
		"normal-big-ambig": X{inv1, SizeSpec{4,4,2}, SizeSpec{5,5,5}, false},
		"normal-toobig":    X{inv1, SizeSpec{7,1,1}, SizeSpec{0,0,0}, true},
		
		"nosmall-micro":     X{inv2, SizeSpec{1,0,0}, SizeSpec{5,1,1}, false},
		"nosmall-small":     X{inv2, SizeSpec{1,1,1}, SizeSpec{5,1,1}, false},
		"nosmall-med-ambig": X{inv2, SizeSpec{3,1,1}, SizeSpec{5,1,1}, false},
		"nosmall-med1":      X{inv2, SizeSpec{4,1,1}, SizeSpec{5,1,1}, false},
		"nosmall-med2":      X{inv2, SizeSpec{2,2,1}, SizeSpec{3,3,1}, false},
		"nosmall-big-ambig": X{inv2, SizeSpec{4,4,2}, SizeSpec{5,5,5}, false},
		"nosmall-toobig":    X{inv2, SizeSpec{7,1,1}, SizeSpec{0,0,0}, true},
		
		"nosmall-swapped-micro":     X{inv3, SizeSpec{1,0,0}, SizeSpec{3,3,1}, false},
		"nosmall-swapped-small":     X{inv3, SizeSpec{1,1,1}, SizeSpec{3,3,1}, false},
		"nosmall-swapped-med-ambig": X{inv3, SizeSpec{3,1,1}, SizeSpec{3,3,1}, false},
		"nosmall-swapped-med1":      X{inv3, SizeSpec{4,1,1}, SizeSpec{5,1,1}, false},
		"nosmall-swapped-med2":      X{inv3, SizeSpec{2,2,1}, SizeSpec{3,3,1}, false},
		"nosmall-swapped-big-ambig": X{inv3, SizeSpec{4,4,2}, SizeSpec{5,5,5}, false},
		"nosmall-swapped-toobig":    X{inv3, SizeSpec{7,1,1}, SizeSpec{0,0,0}, true},
		
		"empty-micro":     X{inv4, SizeSpec{1,0,0}, SizeSpec{1,1,1}, false},
		"empty-small":     X{inv4, SizeSpec{1,1,1}, SizeSpec{1,1,1}, false},
		"empty-med-ambig": X{inv4, SizeSpec{3,1,1}, SizeSpec{0,0,0}, true},
		"empty-med1":      X{inv4, SizeSpec{4,1,1}, SizeSpec{0,0,0}, true},
		"empty-med2":      X{inv4, SizeSpec{2,2,1}, SizeSpec{0,0,0}, true},
		"empty-big-ambig": X{inv4, SizeSpec{4,4,2}, SizeSpec{0,0,0}, true},
		"empty-toobig":    X{inv4, SizeSpec{7,1,1}, SizeSpec{0,0,0}, true},
	}
	
	for k, v := range tests {
		t.Run(k, func(t *testing.T) {
			out, err := v.inv.GetMostSuitableLockerSize(v.size)
			if err != nil && v.is_error {
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %s", err.Error())
				return
			} else if v.is_error {
				t.Errorf("Expected error, but got %v instead", out)
				return
			}
			
			if out != v.inv.Sizes[v.answer] {
				t.Errorf("Wrong answer: expected %v, got %v", v.answer, v.inv.Control[out].Size)
			}
		})
	}	
}
