package dmp

func GetDmpSample() *Dmp {
    return &Dmp{1,2,3,4,5,1,1,1,9,10,
11,2,13,4,5,6,1,
[]uint32{1,2},
[]uint32{2,3},
[]uint32{4,5},
[]uint32{6,7},
[]uint32{8,9},
[]uint32{1,2},
[]uint32{3,4},
[]uint32{1,3},
[]uint32{2,4},
[]uint32{1},
[]uint32{2},
[]uint32{1,4},
[]uint32{2,3},
[]uint32{3},
[]uint32{4},
[]uint32{1},
}
}

func GetDmpAudienceSample() *DmpAudience {
	audience :=&DmpAudience{1,[]uint32{2,1},[]uint32{3,1},[]uint32{4,1},[]uint32{5,1},1,1,1,[]uint32{9},[]uint32{10},
[]uint32{11,1},[]uint32{2,1},[]uint32{3,1},[]uint32{4,1},[]uint32{5,1},[]uint32{6,1},[]uint32{1,2},
[]uint32{1,2},
[]uint32{2,3},
[]uint32{4,5},
[]uint32{6,7},
[]uint32{8,9},
[]uint32{1,2},
[]uint32{3,4},
[]uint32{1,3},
[]uint32{2,4},
[]uint32{1},
[]uint32{2},
[]uint32{1,4},
[]uint32{2,3},
[]uint32{3},
[]uint32{4},
[]uint32{1},
}
	return audience
}
