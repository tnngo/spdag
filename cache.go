package spdag

import (
	"errors"
	"sync"
)

// 轻量级缓存库。

type Cache struct {
	spdagCache map[int]*spdag

	rwmutex *sync.RWMutex
}

// NewCache 初始化缓存库。
func NewCache() *Cache {
	return &Cache{
		spdagCache: make(map[int]*spdag),
		rwmutex:    &sync.RWMutex{},
	}
}

// Add 添加顶点数据至缓存库。
func (c *Cache) Init(key int, vertex *SubbranchPlan) *spdag {
	c.rwmutex.Lock()
	d, ok := c.spdagCache[key]
	if !ok {
		d = &spdag{
			vertexsMap: make(map[int]*SubbranchPlan),
			rwmutex:    &sync.RWMutex{},
		}
		c.spdagCache[key] = d
	}
	d.vertexsMap[vertex.SubbranchPlanId] = vertex
	c.rwmutex.Unlock()
	return d
}

func (c *Cache) Build() {
	for _, ca := range c.spdagCache {
		for _, cv := range ca.vertexsMap {
			ca.Build(cv)
		}
	}
}

// Get 获取key对应的有向无环图。
func (c *Cache) Get(key int) *spdag {
	return c.spdagCache[key]
}

func (c *Cache) GetPlan(vertexID int) *SubbranchPlan {
	for _, d := range c.spdagCache {
		for _, v := range d.vertexsMap {
			if v.SubbranchPlanId == vertexID {
				return v
			}
		}
	}
	return nil
}

func (c *Cache) SpdagList() []*spdag {
	sps := make([]*spdag, 0)
	for _, d := range c.spdagCache {
		sps = append(sps, d)
	}
	return sps
}

func (c *Cache) SPListByPlanIDS(seID int, planIDS []int) ([]*SubbranchPlan, error) {
	v, ok := c.spdagCache[seID]
	if !ok {
		return nil, errors.New("子分部不存在。")
	}

	vertexs := make([]*SubbranchPlan, 0)
	for _, sp := range v.vertexsMap {
		for _, pid := range planIDS {
			if sp.SubbranchPlanId == pid {
				vertexs = append(vertexs, sp)
			}
		}
	}
	return vertexs, nil
}

func (c *Cache) SPListByPlanID(seID int, planID int) (*SubbranchPlan, error) {
	v, ok := c.spdagCache[seID]
	if !ok {
		return nil, errors.New("子分部不存在。")
	}
	pv, pok := v.vertexsMap[planID]
	if !pok {
		return nil, errors.New("计划不存在。")

	}
	return pv, nil
}
