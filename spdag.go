package spdag

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// SubbranchPlan 顶点数据结构。
// 依照安心筑业务扩展顶点数据结构，尽量靠近子分部计划。
type SubbranchPlan struct {
	// SubbranchPlanId 顶点id，唯一标识。在整个有向无环图中必须唯一，通常传入的是数据主键id。
	SubbranchPlanId int `pg:"subbranch_plan_id,pk" json:"subbranch_plan_id"`
	// PreSeq 入度，指向该顶点的顶点集合，可以理解为传入数据的父节点id数组。
	PreSeq []int `pg:",array" json:"pre_seq"`
	// 出度。
	SubSeq []int `pg:",array" json:"sub_seq"`
	// 与入度的最小边长。
	InDegreeMinSideLen time.Duration
	// 与入度的最大时间量。
	InDegreeMaxTime time.Time

	ParentVertexs   []*SubbranchPlan
	ChildrenVertexs []*SubbranchPlan

	/* 以下是子分部计划数据结构SubbranchPlan。*/

	// 子分部计划名称
	SubbranchPlanName string `json:"subbranch_plan_name"`

	// 所属子分部ID
	SubbranchEngId int `json:"subbranch_eng_id"`

	// 所属的单元工程ID
	UnitEngId int `json:"unit_eng_id"`

	// 子分部计划的状态，未开工 =0，施工中=1，已完成=2
	SubbranchPlanStatus int `pg:",use_zero",json:"subbranch_plan_status"`

	// 计划预置开始时间
	PlanStartDate time.Time `json:"plan_start_date"`

	// 计划预置结束时间
	PlanEndDate time.Time `json:"plan_end_date"`

	// 计划实际开始时间
	RealStartDate time.Time `json:"real_start_date"`

	// 计划实际结束时间
	RealEndDate time.Time `json:"real_end_date"`

	// 子分部计划的描述
	SubbranchPlanDesc string `json:"subbranch_plan_desc"`

	// 分部计划的创建时间
	CreatedAt time.Time `json:"created_at"`

	// 分部计划的更新时间
	UpdatedAt time.Time `json:"updated_at"`

	// CompleteTaskCount 已完成班组任务单数量。
	CompleteTaskCount int
}

// 有向无环图数据结构。
type spdag struct {
	// 顶点的数量。
	vertexCount int
	// 有向边的数量。
	edgeCount int

	// 所有顶点。
	vertexsMap map[int]*SubbranchPlan

	rwmutex *sync.RWMutex

	SubbranchEngId int `pg:"subbranch_eng_id,pk" json:"subbranch_eng_id"`
	// 结束时间
	EndDate time.Time `json:"end_date"`
}

// 描绘父节点。
func (d *spdag) drawParent(vertex, parentVertex *SubbranchPlan) {
	has := false
	var tmp time.Time
	if len(vertex.ParentVertexs) == 0 {
		tmp = parentVertex.PlanEndDate
	}
	for _, v := range vertex.ParentVertexs {
		if v.SubbranchPlanId == parentVertex.SubbranchPlanId {
			has = true
		}

		if tmp.IsZero() {
			tmp = v.PlanEndDate
			continue
		}

		if tmp.Before(v.PlanEndDate) {
			tmp = v.PlanEndDate
		}
	}

	// 存储离自己最近的时间。
	vertex.InDegreeMaxTime = tmp

	// 计算边长。
	sub := vertex.PlanStartDate.Sub(tmp)
	vertex.InDegreeMinSideLen = sub

	if !has {
		vertex.ParentVertexs = append(vertex.ParentVertexs, parentVertex)
	}

	hasChildrenID := false
	for _, v := range parentVertex.SubSeq {
		if v == vertex.SubbranchPlanId {
			hasChildrenID = true
		}
	}

	if !hasChildrenID {
		parentVertex.SubSeq = append(parentVertex.SubSeq, vertex.SubbranchPlanId)
	}

}

// 描绘子节点。
func (d *spdag) drawChildren(vertex, childrenVertex *SubbranchPlan) {
	has := false
	for _, v := range childrenVertex.ChildrenVertexs {
		if v.SubbranchPlanId == childrenVertex.SubbranchPlanId {
			has = true
		}
	}

	if !has {
		vertex.ChildrenVertexs = append(vertex.ChildrenVertexs, childrenVertex)
	}

	hasChildrenID := false
	for _, v := range vertex.SubSeq {
		if v == childrenVertex.SubbranchPlanId {
			hasChildrenID = true
		}
	}

	if !hasChildrenID {
		vertex.SubSeq = append(vertex.SubSeq, childrenVertex.SubbranchPlanId)
	}
}

func (d *spdag) updateTime(vertex *SubbranchPlan, sub time.Duration) {
	for _, v := range vertex.ChildrenVertexs {
		v.PlanStartDate = v.PlanStartDate.Add(sub)
		d.updateTime(v, sub)
	}

}

func (d *spdag) allChildrenPlanDate(vertex *SubbranchPlan) {
	for _, v := range vertex.ChildrenVertexs {
		if vertex.PlanEndDate.After(v.InDegreeMaxTime) && !v.InDegreeMaxTime.IsZero() {
			tmp := v.PlanEndDate.Sub(v.PlanStartDate)
			v.PlanStartDate = vertex.PlanEndDate.Add(v.InDegreeMinSideLen)

			v.PlanEndDate = v.PlanStartDate.Add(tmp).Add(v.InDegreeMinSideLen)
			v.InDegreeMaxTime = vertex.PlanEndDate
			d.allChildrenPlanDate(v)
		}
	}
}

func (d *spdag) removeIDSlice(ids []int, index int) []int {
	return append(ids[:index], ids[index+1:]...)
}

// AddRealEndDate 添加实际结束时间。
func (d *spdag) AddRealEndDate(vertexID int, realEndDate time.Time) {
	vertex := d.vertexsMap[vertexID]
	for _, v := range vertex.ChildrenVertexs {
		if realEndDate.After(v.InDegreeMaxTime) && v.InDegreeMaxTime.IsZero() {
			v.PlanStartDate = v.PlanStartDate.Add(v.InDegreeMinSideLen)
			v.PlanEndDate = v.PlanEndDate.Add(v.InDegreeMinSideLen)
			v.InDegreeMaxTime = v.PlanEndDate
			d.allChildrenPlanDate(v)
		}
	}
}

// 画边。
func (d *spdag) drawSide(vertex *SubbranchPlan) {
	for _, ind := range vertex.PreSeq {
		v, ok := d.vertexsMap[ind]
		if ok {
			d.drawParent(vertex, v)
		}
	}

	for _, ind := range vertex.SubSeq {
		if v, ok := d.vertexsMap[ind]; ok {
			vertex.ChildrenVertexs = append(vertex.ChildrenVertexs, v)
		}
	}
}

// 删除vertex切片中的某个数据。
func (d *spdag) removeVertexSlice(vertexs []*SubbranchPlan, index int) []*SubbranchPlan {
	return append(vertexs[:index], vertexs[index+1:]...)
}

// 修改。
func (d *spdag) Update(vertex, oldVertex *SubbranchPlan) (*spdag, error) {
	has := false
	var hasInd int
	for _, ind := range vertex.PreSeq {
		for _, outid := range oldVertex.SubSeq {
			if ind == outid {
				hasInd = ind
				has = true
			}
		}
	}

	if has {
		hasV := d.vertexsMap[hasInd]
		return nil, fmt.Errorf("绘图失败，此操作会导致计划图产生回环。前置计划（%s）已经是（%s）后置计划。", hasV.SubbranchPlanName, vertex.SubbranchPlanName)
	}

	// 判断新传的的预计时间是否会影响到子节点。
	for _, ov := range oldVertex.ChildrenVertexs {
		if vertex.PlanEndDate.After(ov.InDegreeMaxTime) && !ov.InDegreeMaxTime.IsZero() {
			tmp := ov.PlanEndDate.Sub(ov.PlanStartDate)
			ov.PlanStartDate = vertex.PlanEndDate.Add(ov.InDegreeMinSideLen)

			ov.PlanEndDate = ov.PlanStartDate.Add(tmp)
			ov.InDegreeMaxTime = vertex.PlanEndDate
			d.allChildrenPlanDate(ov)
		}
	}

	for _, pv := range oldVertex.ParentVertexs {
		// 删除父节点对应的子节点自己。
		for i, cv := range pv.ChildrenVertexs {
			if cv.SubbranchPlanId == oldVertex.SubbranchPlanId {
				if len(pv.ChildrenVertexs) == 1 {
					pv.ChildrenVertexs = make([]*SubbranchPlan, 0)
				} else {
					pv.ChildrenVertexs = d.removeVertexSlice(pv.ChildrenVertexs, i)
				}
				break
			}
		}

		// 删除父节点对应的子节点自己id。
		for i, cid := range pv.SubSeq {
			if oldVertex.SubbranchPlanId == cid {
				if len(pv.SubSeq) == 1 {
					pv.SubSeq = make([]int, 0)
				} else {
					pv.SubSeq = d.removeIDSlice(pv.SubSeq, i)
					fmt.Println(pv.SubSeq)
				}
				break
			}
		}

	}

	// 重新赋值。
	oldVertex.PreSeq = vertex.PreSeq
	oldVertex.ParentVertexs = make([]*SubbranchPlan, 0)
	oldVertex.PlanStartDate = vertex.PlanStartDate
	oldVertex.PlanEndDate = vertex.PlanEndDate
	oldVertex.RealStartDate = vertex.RealStartDate
	oldVertex.RealEndDate = vertex.RealEndDate
	oldVertex.SubbranchPlanDesc = vertex.SubbranchPlanDesc
	oldVertex.SubbranchPlanName = vertex.SubbranchPlanName
	oldVertex.SubbranchPlanStatus = vertex.SubbranchPlanStatus
	oldVertex.UnitEngId = vertex.UnitEngId

	// 重新绘图。
	d.drawSide(oldVertex)
	for _, ps := range oldVertex.PreSeq {
		pv := d.vertexsMap[ps]
		d.drawChildren(pv, oldVertex)
	}
	return d, nil
}

// 画图。
func (d *spdag) draw(vertex *SubbranchPlan) {
	d.vertexsMap[vertex.SubbranchPlanId] = vertex
	d.drawSide(vertex)
}

// 构建有向无环图。
func (d *spdag) Build(vertex *SubbranchPlan) {
	d.draw(vertex)
}

func (d *spdag) Remove(vertexID int) (map[int]*SubbranchPlan, error) {
	oldVertex, ok := d.vertexsMap[vertexID]
	if !ok {
		return nil, errors.New("未找到计划。")
	}
	for _, ind := range oldVertex.PreSeq {
		if v, ok := d.vertexsMap[ind]; ok {
			// 先删除入度顶点对应的出度数组。
			for i, outID := range v.SubSeq {
				if outID == oldVertex.SubbranchPlanId {
					if len(v.SubSeq) == 1 {
						v.SubSeq = make([]int, 0)
					} else {
						v.SubSeq = d.removeIDSlice(v.SubSeq, i)
					}
					break
				}
			}

			// 再删除子节点vertex数组。
			for i, cv := range v.ChildrenVertexs {
				if cv.SubbranchPlanId == oldVertex.SubbranchPlanId {
					if len(v.ChildrenVertexs) == 1 {
						v.ChildrenVertexs = make([]*SubbranchPlan, 0)
					} else {
						v.ChildrenVertexs = d.removeVertexSlice(v.ChildrenVertexs, i)
					}
					break
				}
			}
		}
	}

	for _, outid := range oldVertex.SubSeq {
		if v, ok := d.vertexsMap[outid]; ok {
			// 先删除出度顶点对应的出度数组。
			for i, outID := range v.SubSeq {
				if outID == oldVertex.SubbranchPlanId {
					if len(v.SubSeq) == 1 {
						v.SubSeq = make([]int, 0)
					} else {
						v.SubSeq = d.removeIDSlice(v.SubSeq, i)
					}
					break
				}
			}

			// 再删除子节点vertex数组。
			for i, pv := range v.ParentVertexs {
				if pv.SubbranchPlanId == oldVertex.SubbranchPlanId {
					if len(pv.ParentVertexs) == 1 {
						pv.ParentVertexs = make([]*SubbranchPlan, 0)
					} else {
						pv.ParentVertexs = d.removeVertexSlice(pv.ParentVertexs, i)
					}
					break
				}
			}
		}

	}
	delete(d.vertexsMap, oldVertex.SubbranchPlanId)
	return d.vertexsMap, nil
}

func (d *spdag) Map() map[int]*SubbranchPlan {
	return d.vertexsMap
}

func (d *spdag) List() []*SubbranchPlan {
	if d == nil {
		return make([]*SubbranchPlan, 0)
	}
	vertexs := make([]*SubbranchPlan, 0)
	for _, v := range d.vertexsMap {
		vertexs = append(vertexs, v)
	}
	return vertexs
}

func (d *spdag) Get(vertexID int) *SubbranchPlan {
	return d.vertexsMap[vertexID]
}

func (d *spdag) delChildrenRepeated(sp []*SubbranchPlan) []*SubbranchPlan {
	newsp := make([]*SubbranchPlan, 0)
	for i := 0; i < len(sp); i++ {
		repeat := false
		for j := i + 1; j < len(sp); j++ {
			if sp[i].SubbranchPlanId == sp[j].SubbranchPlanId {
				repeat = true
				break
			}
		}
		if !repeat {
			newsp = append(newsp, sp[i])
		}
	}
	return newsp
}

func (d *spdag) listChildren(vertex *SubbranchPlan) *SubbranchPlan {
	for _, v := range vertex.ChildrenVertexs {
		return d.listChildren(v)
	}
	return nil
}

// RecursionChildrens 递归当前计划下的所有子节点。
func (d *spdag) RecursionChildrens(vertexID int) []*SubbranchPlan {
	v, ok := d.vertexsMap[vertexID]
	if !ok {
		return nil
	}

	vertexs := make([]*SubbranchPlan, 0)
	cv := d.listChildren(v)
	vertexs = append(vertexs, cv)
	return vertexs
}
