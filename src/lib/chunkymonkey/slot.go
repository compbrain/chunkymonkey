package slot

import (
    "io"
    "os"

    "chunkymonkey/proto"
    .   "chunkymonkey/types"
)

const SlotCountMax = ItemCount(64)

// Represents an inventory slot, e.g in a player's inventory, their cursor, a
// chest.
type Slot struct {
    ItemTypeId ItemTypeId
    Count      ItemCount
    Data       ItemData
}

func (s *Slot) Init() {
    s.ItemTypeId = ItemTypeIdNull
    s.Count = 0
    s.Data = 0
}

func (s *Slot) GetAttr() (ItemTypeId, ItemCount, ItemData) {
    return s.ItemTypeId, s.Count, s.Data
}

func (s *Slot) SendUpdate(writer io.Writer, windowId WindowId, slotId SlotId) os.Error {
    return proto.WriteWindowSetSlot(writer, windowId, slotId, s.ItemTypeId, s.Count, s.Data)
}

func (s *Slot) SendEquipmentUpdate(writer io.Writer, entityId EntityId, slotId SlotId) os.Error {
    return proto.WriteEntityEquipment(writer, entityId, slotId, s.ItemTypeId, s.Data)
}

func (s *Slot) setCount(count ItemCount) {
    s.Count = count
    if s.Count == 0 {
        s.ItemTypeId = ItemTypeIdNull
        s.Data = 0
    }
}

// Adds as many items from the passed slot to the destination (subject) slot as
// possible, depending on stacking allowances and item types etc.
// Returns true if slots changed as a result.
func (s *Slot) Add(src *Slot) (changed bool) {
    // NOTE: This code assumes that 2*SlotCountMax will not overflow
    // the ItemCount type.

    if s.ItemTypeId != ItemTypeIdNull {
        if s.ItemTypeId != src.ItemTypeId {
            return
        }
        if s.Data != src.Data {
            return
        }
    }
    if s.Count >= SlotCountMax {
        return
    }

    s.ItemTypeId = src.ItemTypeId

    toTransfer := src.Count
    if s.Count+toTransfer > SlotCountMax {
        toTransfer = SlotCountMax - s.Count
    }
    if toTransfer != 0 {
        changed = true

        s.Data = src.Data

        s.setCount(s.Count + toTransfer)
        src.setCount(src.Count - toTransfer)
    }
    return
}

// Swaps the contents of the slots.
// Returns true if slots changed as a result.
func (s *Slot) Swap(src *Slot) (changed bool) {
    if s.ItemTypeId != src.ItemTypeId {
        s.ItemTypeId ^= src.ItemTypeId
        src.ItemTypeId ^= s.ItemTypeId
        s.ItemTypeId ^= src.ItemTypeId
        changed = true
    }

    if s.Count != src.Count {
        s.Count ^= src.Count
        src.Count ^= s.Count
        s.Count ^= src.Count
        changed = true
    }

    if s.Data != src.Data {
        s.Data ^= src.Data
        src.Data ^= s.Data
        s.Data ^= src.Data
        changed = true
    }

    return
}

// Splits the contents of the subject slot (s) into half, half remaining in s,
// and half moving to src (odd amounts put the spare item into the src slot).
// If src is not empty, then this does nothing.
// Returns true if slots changed as a result.
func (s *Slot) Split(src *Slot) (changed bool) {
    if s.Count == 0 || src.Count != 0 {
        return
    }

    changed = true
    src.ItemTypeId = s.ItemTypeId
    src.Data = s.Data

    count := s.Count >> 1
    odd := s.Count & 1

    src.Count = count + odd
    s.Count = count

    if s.Count == 0 {
        s.ItemTypeId = ItemTypeIdNull
        s.Data = 0
    }

    return
}

// Takes one item count from src and adds it to the subject s. It does nothing
// if the items in the slots are not compatible.
// Returns true if slots changed as a result.
func (s *Slot) AddOne(src *Slot) (changed bool) {
    if src.Count == 0 || s.Count >= SlotCountMax {
        return
    }
    if src.Data != s.Data {
        return
    }
    if s.ItemTypeId != src.ItemTypeId && s.ItemTypeId != ItemTypeIdNull {
        return
    }

    changed = true
    s.Count++
    src.Count--

    s.ItemTypeId = src.ItemTypeId
    s.Data = src.Data

    if src.Count == 0 {
        src.ItemTypeId = ItemTypeIdNull
        src.Data = 0
    }

    return
}