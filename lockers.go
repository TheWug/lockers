package lockers

import (
	"errors"
	
	"github.com/google/uuid"
)

type LockerSize int

type IControlSpec interface{
	ControlSpec(LockerSize) *LockerControlSpec
}

// perform an inventory aware comparison of 2 locker sizes.
// the "lowest" locker size is the one with the largest number of
// available lockers, and then the one with the smallest volume.
func (id LockerSize) Before(other_id LockerSize, inv IControlSpec) bool {
	self, other := inv.ControlSpec(id), inv.ControlSpec(other_id)
	if self.VirtualCapacity > other.VirtualCapacity {
		return true
	} else if self.VirtualCapacity == other.VirtualCapacity && self.Size.Volume() < other.Size.Volume() {
		return true
	}
	
	return false
}

type SizeSpec struct {
	Length, Width, Height int
}

func (spec SizeSpec) Normalize() SizeSpec {
	// make sure all values are positive or zero
	// this doesn't actually completely work, because MIN_INT = -MIN_INT, but
	// I'm ignoring that case because it is not practically useful to consider it
	if spec.Length < 0 {
		spec.Length = 0 - spec.Length
	}
	if spec.Width < 0 {
		spec.Width = 0 - spec.Width
	}
	if spec.Height < 0 {
		spec.Height = 0 - spec.Height
	}
	
	// hard coded 3 element bubble sort
	if spec.Height > spec.Width {
		spec.Height, spec.Width = spec.Width, spec.Height
	}
	if spec.Width > spec.Length {
		spec.Width, spec.Length = spec.Length, spec.Width
	}
	if spec.Height > spec.Width {
		spec.Height, spec.Width = spec.Width, spec.Height
	}
	return spec
}

func (spec SizeSpec) Volume() int64 {
	return int64(spec.Length) * int64(spec.Width) * int64(spec.Height)
}

// expects normalized SizeSpecs
func (spec SizeSpec) Contains(other SizeSpec) bool {
	return spec.Length >= other.Length &&
	       spec.Width  >= other.Width  &&
	       spec.Height >= other.Height
}

type LockerControlSpec struct {
	SizeId LockerSize
	Size SizeSpec
	
	BiggerThan []LockerSize
	SmallerThan []LockerSize
	
	Lockers []int
	
	VirtualCapacity int
}

func (lcs LockerControlSpec) Full() bool {
	return len(lcs.Lockers) == 0
}

type Locker struct {
	Id string
	SizeId LockerSize
	
	Contents *Package
}

type Package struct {
	Id string
	Size SizeSpec
	
	StoredIn *Locker
}

type Inventory struct {
	Lockers []Locker
	
	Control map[LockerSize]*LockerControlSpec
	Sizes map[SizeSpec]LockerSize
	
	LockersById map[string]int
	LockersByPackageId map[string]int
}

func (inv Inventory) ControlSpec(size_id LockerSize) *LockerControlSpec {
	return inv.Control[size_id]
}

func (l *Locker) Put(pkg *Package) error {
	if l.Contents != nil {
		return errors.New("Locker is not empty")
	} else if pkg.StoredIn != nil {
		return errors.New("Package already in locker")
	}
	
	l.Contents = pkg
	pkg.StoredIn = l
	return nil
}

func (l *Locker) Fetch() (*Package, error) {
	if l.Contents == nil {
		return nil, errors.New("Tried to fetch from empty locker")
	}
	
	p := l.Contents
	l.Contents = nil
	p.StoredIn = nil
	return p, nil
}

func NewInventory(locker_counts_by_size map[SizeSpec]int) *Inventory {
	total_locker_count := 0
	for _, count := range locker_counts_by_size {
		total_locker_count += count
	}
	
	inv := &Inventory{
		Control: make(map[LockerSize]*LockerControlSpec, len(locker_counts_by_size)),
		Sizes: make(map[SizeSpec]LockerSize, len(locker_counts_by_size)),
		
		LockersById: make(map[string]int, total_locker_count),
		LockersByPackageId: make(map[string]int, total_locker_count),
		
		Lockers: make([]Locker, total_locker_count, total_locker_count),
	}

	// normalize the sizes and allocate a LockerSize for each,
	// and build the master locker list. Locker "pointers" are just
	// indices into this array.
	// O(n + L) for L lockers of n distinct sizes.
	sizes := make([]SizeSpec, 0, len(locker_counts_by_size))
	index := 0
	for size, count := range locker_counts_by_size {
		size = size.Normalize()
		var size_id LockerSize
		var ok bool
		if size_id, ok = inv.Sizes[size]; !ok {
			sizes = append(sizes, size)
			size_id = LockerSize(len(inv.Sizes) + 1)
			inv.Sizes[size] = size_id
			inv.Control[size_id] = &LockerControlSpec{
				SizeId: size_id,
				Size: size,
				Lockers: make([]int, 0, count),
			}
		}
		
		for max, i := count + index, 0; index < max; index, i = index+1, i+1 {
			id := uuid.NewString()
			inv.Lockers[index] = Locker{
				SizeId: size_id,
				Id: id,
			}
			inv.LockersById[id] = index
			inv.Control[size_id].Lockers = append(inv.Control[size_id].Lockers, index)
		}
	}
	
	// for each size, compute which other sizes fit entirely within it
	// and store a bidirectional graph representing this relationship.
	// runs in O(n^2) for n distinct sizes, which is not too bad given n's
	// tendancy to be fairly small
	for i, s1 := range sizes {
		for _, s2 := range sizes[i+1:] {
			if s1.Contains(s2) {
				inv.Control[inv.Sizes[s1]].BiggerThan  = append(inv.Control[inv.Sizes[s1]].BiggerThan,  inv.Sizes[s2])
				inv.Control[inv.Sizes[s2]].SmallerThan = append(inv.Control[inv.Sizes[s2]].SmallerThan, inv.Sizes[s1])
			} else if s2.Contains(s1) {
				inv.Control[inv.Sizes[s1]].SmallerThan = append(inv.Control[inv.Sizes[s1]].SmallerThan, inv.Sizes[s2])
				inv.Control[inv.Sizes[s2]].BiggerThan  = append(inv.Control[inv.Sizes[s2]].BiggerThan,  inv.Sizes[s1])
			}
		}
	}
	
	// calculate the virtual capacity of each locker group
	// this also runs in O(n^2) time. The problem is finding partial
	// sums of nodes in a directed acyclig graph. Somewhat to my surprise,
	// there is no known algorithm which does this in better than O(n^2).
	// again though, n is likely to be fairly small.
	for _, ctrl := range inv.Control {
		for _, other_id := range ctrl.SmallerThan {
			ctrl.VirtualCapacity += len(inv.Control[other_id].Lockers)
		}
		ctrl.VirtualCapacity += len(ctrl.Lockers)
	}
	
	return inv
}

func (inv *Inventory) GetMostSuitableLockerSize(package_size SizeSpec) (LockerSize, error) {
	// build a list of all locker sizes which a. have empty lockers and
	// b. have enough space for the given dimensions
	candidate_sizes := make([]LockerSize, 0, len(inv.Sizes))
	for size, size_id := range inv.Sizes {
		if !size.Contains(package_size) { continue }
		if inv.Control[size_id].Full() { continue }
		
		candidate_sizes = append(candidate_sizes, size_id)
	}
	
	if len(candidate_sizes) == 0 {
		return LockerSize(0), errors.New("No available lockers which can fit package")
	}
	
	// choose the most eligible candidate
	chosen_id := candidate_sizes[0]
	for _, id := range candidate_sizes[1:] {
		if id.Before(chosen_id, inv) {
			chosen_id = id
		}
	}

	return chosen_id, nil
}

func (inv *Inventory) DepositPackage(pkg *Package) (string, error) {
	if _, ok := inv.LockersByPackageId[pkg.Id]; ok {
		return "", errors.New("Duplicate package ID")
	}

	chosen_id, err := inv.GetMostSuitableLockerSize(pkg.Size.Normalize())
	if err != nil {
		return "", err
	}
	
	locker_index := inv.AllocateLocker(chosen_id)
	inv.Lockers[locker_index].Put(pkg)
	return inv.Lockers[locker_index].Id, nil
}

func (inv *Inventory) RetrievePackage(pkg *Package) (*Package, error) {
	return inv.RetrievePackageById(pkg.Id)
}

func (inv *Inventory) RetrievePackageById(id string) (*Package, error) {
	lid, ok := inv.LockersByPackageId[id]
	return inv.RetrievePackageInternal(lid, ok)
}

func (inv *Inventory) RetrievePackageByLockerId(id string) (*Package, error) {
	lid, ok := inv.LockersById[id]
	return inv.RetrievePackageInternal(lid, ok)
}

func (inv *Inventory) RetrievePackageInternal(locker_index int, ok bool) (*Package, error) {
	if !ok {
		return nil, errors.New("Package ID not known")
	}
	
	pkg, err := inv.Lockers[locker_index].Fetch()
	if err != nil {
		return nil, err
	}
	
	
	inv.DeallocateLocker(locker_index)
	return pkg, nil
}

func (inv *Inventory) AllocateLocker(size_id LockerSize) int {
	ctrl := inv.Control[size_id]
	locker_index := ctrl.Lockers[len(ctrl.Lockers) - 1]
	ctrl.Lockers = ctrl.Lockers[:len(ctrl.Lockers) - 1]
	inv.AdjustVirtualCapacity(size_id, -1)
	return locker_index
}

func (inv *Inventory) DeallocateLocker(locker_index int) {
	size_id := inv.Lockers[locker_index].SizeId
	inv.Control[size_id].Lockers = append(inv.Control[size_id].Lockers, locker_index)
	inv.AdjustVirtualCapacity(size_id, 1)
}

func (inv *Inventory) AdjustVirtualCapacity(size_id LockerSize, by int) {
	inv.Control[size_id].VirtualCapacity += by
	for _, other_id := range inv.Control[size_id].BiggerThan {
		inv.Control[other_id].VirtualCapacity += by
	}
}
