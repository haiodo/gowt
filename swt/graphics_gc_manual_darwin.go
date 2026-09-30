// GC.GCTextData.Key/Cache (manual.txt): a record key and a size-bounded LinkedHashMap whose
// removeEldestEntry releases the evicted text layout.
package swt

import "unsafe"

type GC_GCTextData_Key struct {
	string_ string
	alpha   int32
	font    int64
	// Java record equality on an array component is reference identity.
	fgColor *float64
}

func NewGC_GCTextData_Key(s string, alpha int32, font int64, fgColor []float64) *GC_GCTextData_Key {
	return &GC_GCTextData_Key{s, alpha, font, unsafe.SliceData(fgColor)}
}

type GC_GCTextData_Cache struct {
	cacheSize int32
	keys      []GC_GCTextData_Key
	cache     map[GC_GCTextData_Key]*GC_GCTextData
}

func NewGC_GCTextData_Cache(cacheSize int32) *GC_GCTextData_Cache {
	return &GC_GCTextData_Cache{cacheSize: cacheSize, cache: map[GC_GCTextData_Key]*GC_GCTextData{}}
}

func (c *GC_GCTextData_Cache) Release() {
	for _, data := range c.cache {
		data.Release()
	}
	c.keys = nil
	c.cache = map[GC_GCTextData_Key]*GC_GCTextData{}
}

func (c *GC_GCTextData_Cache) Get(key *GC_GCTextData_Key) *GC_GCTextData { return c.cache[*key] }

func (c *GC_GCTextData_Cache) Put(key *GC_GCTextData_Key, data *GC_GCTextData) {
	if _, ok := c.cache[*key]; !ok {
		c.keys = append(c.keys, *key)
	}
	c.cache[*key] = data
	if int32(len(c.keys)) >= c.cacheSize { // removeEldestEntry: size() >= cacheSize
		c.cache[c.keys[0]].Release()
		delete(c.cache, c.keys[0])
		c.keys = c.keys[1:]
	}
}
