package mysql

// coverageNode record tcp package begin and end seq id
type coverageNode struct {
	begin int64
	end   int64

	next *coverageNode
	crp  *coveragePool
}

func newCoverage(begin, end int64) (*coverageNode) {
	return &coverageNode{
		begin: begin,
		end:   end,
	}
}

func (crn *coverageNode) Recovery() {
	crn.crp.Enqueue(crn)
}

type coverRanges struct {
	head *coverageNode
}

func NewCoverRanges() *coverRanges {
	return &coverRanges{
		head: &coverageNode{
			begin: -1,
			end:   -1,
		},
	}
}

func (crs *coverRanges) clear() {
	currRange := crs.head.next;
	for currRange != nil {
		node := currRange
		currRange = currRange.next
		node.Recovery()
	}
	crs.head.next = nil
}

func (crs *coverRanges) addRange(node *coverageNode) {
	// insert range in asc order
	var currRange = crs.head;
	for currRange != nil && currRange.next != nil {
		checkRange := currRange.next
		if checkRange != nil && checkRange.begin >= node.begin {
			currRange.next = node
			node.next = checkRange
			node = nil
			break
		}

		currRange = checkRange
	}

	if node != nil && currRange != nil {
		currRange.next = node
	}

	crs.mergeRanges()
}

func (crs *coverRanges) mergeRanges() {
	// merge ranges
	currRange := crs.head.next
	for currRange != nil && currRange.next != nil {
		checkRange := currRange.next
		if currRange.begin <= checkRange.begin && currRange.end >= checkRange.begin && currRange.end < checkRange.end {
			currRange.end = checkRange.end
			currRange.next = checkRange.next
			checkRange.Recovery()

		} else {
			currRange = currRange.next
		}
	}
}

type coveragePool struct {
	queue chan *coverageNode
}

func NewCoveragePool() (cp *coveragePool) {
	return &coveragePool{
		queue: make(chan *coverageNode, 256),
	}
}

func (crp *coveragePool) NewCoverage(begin, end int64) (cn *coverageNode) {
	cn = crp.Dequeue()
	cn.begin = begin
	cn.end = end
	return
}

func (crp *coveragePool) Enqueue(cn *coverageNode) {
	// log.Debugf("coveragePool enqueue: %d", len(crp.queue))
	if cn == nil {
		return
	}

	select {
	case crp.queue <- cn:
		return

	default:
		cn = nil
	}
}

func (crp *coveragePool) Dequeue() (cn *coverageNode) {
	// log.Debugf("coveragePool dequeue: %d", len(crp.queue))

	defer func() {
		cn.begin = -1
		cn.end = -1
		cn.next = nil
		cn.crp = crp
	}()

	select {
	case cn = <-crp.queue:
		return

	default:
		cn = &coverageNode{}
		return
	}
}
