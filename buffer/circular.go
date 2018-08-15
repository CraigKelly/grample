package buffer

// CircularInt is a circular buffer of ints with the ability to iterate over
// the first and second halves of the integers collected in the order that they
// were appended.
type CircularInt struct {
	buffer    []int // actual storage
	pos       int   // Current position in buffer
	BufSize   int   // BufSize is the fixed number of ints maintained in memory
	Count     int   // Count is the number of ints in memory. Will always be <= BufSize
	TotalSeen int64 // TotalSeen is the total number of times Add has been called
}

// NewCircularInt creates a new circular buffer of totalSize. If totalSize is
// not a multiple of 2, it will be adjusted.
func NewCircularInt(totalSize int) *CircularInt {
	// Fix odd number situations
	half := totalSize / 2
	total := half + half

	return &CircularInt{
		buffer:  make([]int, total),
		pos:     0,
		BufSize: total,
		Count:   0,
	}
}

// Internal: return the next array position
func (c *CircularInt) nextPos() int {
	return (c.pos + 1) % c.BufSize
}

// Add appends the given int to the buffer, overwriting the oldest entry
func (c *CircularInt) Add(i int) error {
	c.TotalSeen++

	c.buffer[c.pos] = i

	c.pos = c.nextPos()

	c.Count++
	if c.Count > c.BufSize {
		c.Count = c.BufSize // max out
	}

	return nil
}

// FirstHalf returns an iterator over the first (oldest) half of the stored
// values. Will not return a valid iterator until Add has been called at least
// BufSize times
func (c *CircularInt) FirstHalf() *CircularIntIterator {
	if c.Count < c.BufSize {
		return nil
	}

	return &CircularIntIterator{
		buf:    c,
		curr:   c.pos, // Oldest is the one we're about to write
		remain: c.BufSize / 2,
	}
}

// SecondHalf returns an iterator over the second (most recent) half of the
// stored values. Will not return a valid iterator until Add has been called at
// least BufSize times
func (c *CircularInt) SecondHalf() *CircularIntIterator {
	if c.Count < c.BufSize {
		return nil
	}

	half := c.BufSize / 2
	pos := (c.pos + half) % c.BufSize

	return &CircularIntIterator{
		buf:    c,
		curr:   pos,
		remain: half,
	}
}

// CircularIntIterator provides an iterator over a CircularInt buffer
type CircularIntIterator struct {
	buf    *CircularInt
	curr   int
	remain int
}

// Next returns True when there are more values to read via Value
func (i *CircularIntIterator) Next() bool {
	return i.remain > 0
}

// Value return the next integer to be read. Should only be called if Next() is
// True
func (i *CircularIntIterator) Value() int {
	v := i.buf.buffer[i.curr]
	i.curr = (i.curr + 1) % i.buf.BufSize
	i.remain--
	return v
}
