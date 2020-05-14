package spdag

import (
	"testing"
	"time"
)

func TestNewCache(t *testing.T) {
	t.Run("缓存库初始化。", func(t *testing.T) {
		//jtime1 := "2015-03-20 08:50:29"
		//time2 := "2015-03-21 09:04:25"
		time3 := "2015-03-30 09:04:25"
		//time4 := "2015-05-01 09:04:25"
		//t1, _ := time.Parse("2006-01-02 15:04:05", time1)
		//t2, _ := time.Parse("2006-01-02 15:04:05", time2)
		t3, _ := time.Parse("2006-01-02 15:04:05", time3)
		//t4, _ := time.Parse("2006-01-02 15:04:05", time4)
		c := NewCache()
		v1 := &SubbranchPlan{
			SubbranchPlanId:   1,
			SubbranchPlanName: "123",
			SubSeq:            []int{2, 3},
		}

		v2 := &SubbranchPlan{
			SubbranchPlanId:   2,
			SubbranchPlanName: "789",
			PreSeq:            []int{1},
			SubSeq:            []int{3},
		}
		v3 := &SubbranchPlan{
			SubbranchPlanId:   3,
			SubbranchPlanName: "111",
			PlanStartDate:     t3,
			PreSeq:            []int{1, 2},
		}

		c.Init(111, v1)
		c.Init(111, v3)
		c.Init(111, v2)
		c.Build()

		dag := c.Get(111)
		v4 := &SubbranchPlan{
			SubbranchPlanId:   3,
			SubbranchPlanName: "111",
			PlanStartDate:     t3,
			PreSeq:            []int{1},
		}
		aa := dag.Get(3)
		dag.Update(v4, aa)

		//c.Add(111, v)
		//c.Add(111, v3)
		//c.Add(111, v7)
		//c.Add(111, v5)
		//c.Add(111, v6)
		//d := c.Add(111, v10)
		//t.Logf("结果：%+v\n\n", d.vertexsMap[123])
		//d.Update(v3, aa)
		for _, v := range c.SpdagList() {
			for _, p := range v.vertexsMap {

				t.Logf("结果：%+v\n\n", p)
			}
		}
	})
}

func Test_cache_Get(t *testing.T) {
	//t.Run("cache.Get", func(t *testing.T) {
	//	c := NewCache()
	//	vertex := &SubbranchPlan{
	//		ID: 123,
	//	}
	//	c.Add(1, vertex)
	//	d := c.Get(1)
	//	t.Log(d)
	//})
}
