// Structures and algorithms for efficiently managing the logistics
// of a system of lockers and packages, generalized to 3D rectangular packages
// and lockers of any collection of sizes.
// Inserting a package is O(n), for n number of different SIZES of LOCKER.
package lockers

import (
	"errors"

	"github.com/google/uuid"
)

// defines a unique ID which corresponds to a particular size of locker.
type LockerSize int

// defines IDs for identifying packages. Expected to be unique, in an inventory.
type PackageID string

// defines IDs for identifying lockers. Expected to be unique, in an inventory.
type LockerID string

// defines a limited interface by which LockerSize objects might access Inventory
// functionality when determining relative priority for storing new objects.
type IControlSpec interface{
	ControlSpec(LockerSize) *LockerControlSpec
}

// Performs an inventory aware comparison of 2 locker sizes. the "earliest"
// locker size is the one with the largest number of available spaces for
// items of this size, and then the one with the smallest volume.
func (id LockerSize) Before(other_id LockerSize, inv IControlSpec) bool {
	self, other := inv.ControlSpec(id), inv.ControlSpec(other_id)
	if self.VirtualCapacity > other.VirtualCapacity {
		return true
	} else if self.VirtualCapacity == other.VirtualCapacity && self.Size.Volume() < other.Size.Volume() {
		return true
	}

	return false
}

// a 3 dimensional vector, concretely representing the dimensions of a locker or package.
type SizeSpec struct {
	Length, Width, Height int
}

// Normalizes a SizeSpec by making any negative dimensions positive, and then sorting
// dimensions in descending order.
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

// Computes the 3D volume of a SizeSpec, length * width * height.
func (spec SizeSpec) Volume() int64 {
	return int64(spec.Length) * int64(spec.Width) * int64(spec.Height)
}

// Checks if a SizeSpec fully contains another.  You MUST normalize both SizeSpecs
// before using this function, or it will produce inaccurate results.
func (spec SizeSpec) Contains(other SizeSpec) bool {
	return spec.Length >= other.Length &&
	       spec.Width  >= other.Width  &&
	       spec.Height >= other.Height
}

// An internal structure which represents a collection of lockers of a single size.
// Contains lists of other locker sizes which are bigger/smaller, as well as
// the combined total free capacity of all lockers which are equal or larger.
type LockerControlSpec struct {
	SizeId LockerSize
	Size SizeSpec

	BiggerThan []LockerSize
	SmallerThan []LockerSize

	Lockers []int

	VirtualCapacity int
}

// Returns true if a LockerControlSpec has no available lockers and false otherwise.
func (lcs LockerControlSpec) Full() bool {
	return len(lcs.Lockers) == 0
}

// A structure which represents a locker. Lockers come in discrete sizes.
type Locker struct {
	Id LockerID
	SizeId LockerSize

	Contents *Package
}

// A structure which represents a package. Packages can come in any size.
type Package struct {
	Id PackageID
	Size SizeSpec

	StoredIn *Locker
}

// The inventory structure manages what lockers are available and what packages
// they contain. This provides the primary functionality of this module.
type Inventory struct {
	Lockers []Locker

	Control map[LockerSize]*LockerControlSpec
	Sizes map[SizeSpec]LockerSize

	LockersById map[LockerID]int
	LockersByPackageId map[PackageID]int
}

// Fetches the locker control group of requested size, or nil if none exists.
// required to implement IControlSpec.
func (inv Inventory) ControlSpec(size_id LockerSize) *LockerControlSpec {
	return inv.Control[size_id]
}

// Puts a package into a locker. Returns an error if there is a problem, such as
// a locker which already has an item in it or a package which is already in a
// locker, or nil if the operation completes normally.
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

// Fetches an item from a locker.  Returns nil and an error if the locker is
// empty, or the package and nil otherwise.
func (l *Locker) Fetch() (*Package, error) {
	if l.Contents == nil {
		return nil, errors.New("Tried to fetch from empty locker")
	}

	p := l.Contents
	l.Contents = nil
	p.StoredIn = nil
	return p, nil
}

// Creates a new inventory.
// Pass it a map, with desired locker dimensions as keys and locker counts as values.
// Denormalized and even duplicate values are permitted and will be handled gracefully
// (e.g. {{1,2,3}:5, {3,2,1}:5} is equivalent to {{3,2,1}:10}). Non-duplicate values do
// carry the performance optimization of exactly sizing some data structures, so they
// are preferred if possible.  Empty inventories are allowed, though they are not useful.
// Adding lockers/locker sizes to an inventory on the fly is possible but unimplemented.
// Removing lockers is not an easy prospect, but is possible by making some changes to
// how available lockers are stored.
func NewInventory(locker_counts_by_size map[SizeSpec]int) *Inventory {
	total_locker_count := 0
	for _, count := range locker_counts_by_size {
		total_locker_count += count
	}

	inv := &Inventory{
		Control: make(map[LockerSize]*LockerControlSpec, len(locker_counts_by_size)),
		Sizes: make(map[SizeSpec]LockerSize, len(locker_counts_by_size)),

		LockersById: make(map[LockerID]int, total_locker_count),
		LockersByPackageId: make(map[PackageID]int, total_locker_count),

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
			id := LockerID(uuid.NewString())
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

// Fetches the most appropriate size of locker to store a given size of package in.
// This is defined to be the size class of locker with the largest available capacity
// in terms of both direct storage, and also larger available lockers.
// If multiple such options exist, the smallest (volume-wise) locker is chosen.
// note: this is usually, but not always, the locker with the best space efficiency.
// An example where it is not:
// imagine an inventory with 3 sizes of locker:
// small-1 (4x1x1, 1   available)
// small-2 (2x2x2, 100 available)
// medium  (4x4x4, 1   available)
// small-1 cannot fit into small-2, and neither can small-2 fit into small-1,
// but both can fit inside medium.
// if a 2x1x1 package comes in, it will be placed into small-2 even though it would be more
// space efficient to place it into small-1, because of the relative scarcity of lockers
// large enough to hold a package which could fit into small-1 but not small-2.
// I assert that a space-optimizing algorithm would lead you astray if you applied it here.
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

// places a package into the inventory. O(n) for n different size lockers.
// returns a locker ID and nil, or "" and an error if one occurs.
func (inv *Inventory) DepositPackage(pkg *Package) (LockerID, error) {
	if _, ok := inv.LockersByPackageId[pkg.Id]; ok {
		return "", errors.New("Duplicate package ID")
	}

	chosen_id, err := inv.GetMostSuitableLockerSize(pkg.Size.Normalize())
	if err != nil {
		return "", err
	}

	ctrl := inv.Control[chosen_id]
	locker_index := ctrl.Lockers[len(ctrl.Lockers) - 1]
	err = inv.Lockers[locker_index].Put(pkg)
	if err != nil {
		return "", err
	}

	inv.AllocateLocker(chosen_id)
	inv.LockersByPackageId[pkg.Id] = locker_index
	return inv.Lockers[locker_index].Id, nil
}

// removes a package from the inventory (via package ID lookup).
func (inv *Inventory) RetrievePackage(pkg *Package) (*Package, error) {
	return inv.RetrievePackageById(pkg.Id)
}

// removes a package from the inventory.
func (inv *Inventory) RetrievePackageById(id PackageID) (*Package, error) {
	lid, ok := inv.LockersByPackageId[id]
	return inv.RetrievePackageInternal(lid, ok)
}

// removes a package from the inventory.
func (inv *Inventory) RetrievePackageByLockerId(id LockerID) (*Package, error) {
	lid, ok := inv.LockersById[id]
	return inv.RetrievePackageInternal(lid, ok)
}

// retrieves a package from the inventory. O(n) for n different size lockers.
// internal function, not meant to be called directly.
func (inv *Inventory) RetrievePackageInternal(locker_index int, ok bool) (*Package, error) {
	if !ok {
		return nil, errors.New("Package ID not known")
	}

	pkg, err := inv.Lockers[locker_index].Fetch()
	if err != nil {
		return nil, err
	}

	inv.DeallocateLocker(locker_index)
	delete(inv.LockersByPackageId, pkg.Id)
	return pkg, nil
}

// Reserves a locker of the given size. This immediately removes it from the
// available lockers in the inventory, and updates the inventory's space availability
func (inv *Inventory) AllocateLocker(size_id LockerSize) int {
	ctrl := inv.Control[size_id]
	locker_index := ctrl.Lockers[len(ctrl.Lockers) - 1]
	ctrl.Lockers = ctrl.Lockers[:len(ctrl.Lockers) - 1]
	inv.AdjustVirtualCapacity(size_id, -1)
	return locker_index
}

// returns a locker to the inventory. This immediately returns it to the inventory's
// pool of available lockers and updates the inventory's space availability.
func (inv *Inventory) DeallocateLocker(locker_index int) {
	size_id := inv.Lockers[locker_index].SizeId
	inv.Control[size_id].Lockers = append(inv.Control[size_id].Lockers, locker_index)
	inv.AdjustVirtualCapacity(size_id, 1)
}

// Updates the inventory's space availability by adding the specified amount to
// the given locker size, and all other lockers large enough to hold the same contents
func (inv *Inventory) AdjustVirtualCapacity(size_id LockerSize, by int) {
	inv.Control[size_id].VirtualCapacity += by
	for _, other_id := range inv.Control[size_id].BiggerThan {
		inv.Control[other_id].VirtualCapacity += by
	}
}
