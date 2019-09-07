package mysql

type handshakeResponse41 struct {
	Capability uint32
	Collation  uint8
	User       string
	DBName     string
	Auth       []byte
}

// coverageNode record tcp package begin and end seq id
type coverageNode struct {
	begin int64
	end   int64

	next *coverageNode
	crp *coveragePool
}

func newCoverage(begin, end int64) (*coverageNode) {
	return &coverageNode{
		begin: begin,
		end: end,
	}
}

func (crn *coverageNode) Recovery()  {
	crn.crp.Enqueue(crn)
}

type coveragePool struct {
	queue  []*coverageNode
}


func NewCoveragePool() (cp *coveragePool) {
	return &coveragePool{
		queue: make([]*coverageNode, 0, 256),
	}
}

func (crp *coveragePool) Enqueue(cn *coverageNode)  {
	if cn == nil {
		return
	}

	crp.queue = append(crp.queue, cn)
}

func (crp *coveragePool) NewCoverage(begin, end int64)(cn *coverageNode)  {
	cn = crp.Dequeue()
	cn.begin = begin
	cn.end = end
	return
}

func (crp *coveragePool) Dequeue() (cn *coverageNode)  {
	defer func() {
		cn.begin = -1
		cn.end = -1
		cn.next = nil
		cn.crp = crp
	}()

	if len(crp.queue) < 1 {
		cn = &coverageNode{}
		return
	}

	cn = crp.queue[0]
	crp.queue = crp.queue[1:]
	return
}

type coverRanges struct {
	head *coverageNode
}

func NewCoverRanges() *coverRanges {
	return &coverRanges{}
}

func (crs *coverRanges) clear() {
	if crs.head == nil {
		return
	}

	currRange := crs.head;
	if currRange.next != nil {
		node := currRange
		currRange = currRange.next
		node.Recovery()
	}
	crs.head = nil
}

func (crs *coverRanges) addRange(node *coverageNode)  {
	// empty cover ranges
	if crs.head == nil {
		crs.head = node
		return
	}

	// insert range in asc order
	var currRange = crs.head;
	for currRange.next != nil {
		checkRange := currRange.next
		if checkRange.begin >= node.begin {
			currRange.next = node
			node.next = checkRange
			node = nil
			break
		}
	}
	if node != nil {
		currRange.next = node
	}

	crs.mergeRanges()
}

func (crs *coverRanges) mergeRanges()  {
	// merge ranges
	currRange := crs.head;
	if currRange.next != nil {
		checkRange := currRange.next
		if currRange.end >= checkRange.begin && currRange.end < checkRange.end {
			currRange.end = checkRange.end
			currRange.next = checkRange.next

			checkRange.Recovery()
		}
	}
}