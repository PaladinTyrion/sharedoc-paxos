package "main"

type PadManager struct {
  PadId     int64
  CurrRev   uint64
  OpHistory map[uint64]Op // revision base -> committed Op
}
