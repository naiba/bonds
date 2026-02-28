package services

import (
	"container/heap"
	"errors"
	"math"
	"strings"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
	"gorm.io/gorm"
)

var ErrRelationshipNotFound = errors.New("relationship not found")

type RelationshipService struct {
	db           *gorm.DB
	feedRecorder *FeedRecorder
}

func NewRelationshipService(db *gorm.DB) *RelationshipService {
	return &RelationshipService{db: db}
}

func (s *RelationshipService) SetFeedRecorder(fr *FeedRecorder) {
	s.feedRecorder = fr
}

func (s *RelationshipService) List(contactID, vaultID string) ([]dto.RelationshipResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	var relationships []models.Relationship
	// BUG FIX: Query both directions so that relationships appear on both
	// participants' pages, not just the creator's. Previously only queried
	// contact_id = ?, which caused reverse-created records to be invisible.
	if err := s.db.Preload("RelationshipType").
		Where("contact_id = ? OR related_contact_id = ?", contactID, contactID).
		Order("created_at DESC").Find(&relationships).Error; err != nil {
		return nil, err
	}
	result := make([]dto.RelationshipResponse, 0, len(relationships))
	for _, r := range relationships {
		resp := toRelationshipResponse(&r)
		// For reverse relationships (where this contact is the related_contact),
		// swap IDs so the "related" contact in the response is always the OTHER person,
		// and show the reverse type name.
		if r.RelatedContactID == contactID {
			resp.ContactID = contactID
			resp.RelatedContactID = r.ContactID
			if r.RelationshipType.NameReverseRelationship != nil {
				resp.RelationshipTypeName = *r.RelationshipType.NameReverseRelationship
			}
		}
		result = append(result, resp)
	}
	return result, nil
}

func (s *RelationshipService) Create(contactID, vaultID string, req dto.CreateRelationshipRequest) (*dto.RelationshipResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}

	var relationship models.Relationship
	err := s.db.Transaction(func(tx *gorm.DB) error {
		relationship = models.Relationship{
			ContactID:          contactID,
			RelationshipTypeID: req.RelationshipTypeID,
			RelatedContactID:   req.RelatedContactID,
		}
		if err := tx.Create(&relationship).Error; err != nil {
			return err
		}

		// Auto-create reverse relationship
		reverseTypeID, found := findReverseTypeID(tx, req.RelationshipTypeID)
		if found {
			reverse := models.Relationship{
				ContactID:          req.RelatedContactID,
				RelationshipTypeID: reverseTypeID,
				RelatedContactID:   contactID,
			}
			if err := tx.Create(&reverse).Error; err != nil {
				return err
			}
			if s.feedRecorder != nil {
				entityType := "Relationship"
				s.feedRecorder.Record(req.RelatedContactID, "", ActionRelationshipAdded, "Added a relationship", &reverse.ID, &entityType)
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	if s.feedRecorder != nil {
		entityType := "Relationship"
		s.feedRecorder.Record(contactID, "", ActionRelationshipAdded, "Added a relationship", &relationship.ID, &entityType)
	}

	resp := toRelationshipResponse(&relationship)
	return &resp, nil
}

func (s *RelationshipService) Update(id uint, contactID, vaultID string, req dto.UpdateRelationshipRequest) (*dto.RelationshipResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}
	var relationship models.Relationship
	if err := s.db.Where("id = ? AND contact_id = ?", id, contactID).First(&relationship).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRelationshipNotFound
		}
		return nil, err
	}
	relationship.RelationshipTypeID = req.RelationshipTypeID
	relationship.RelatedContactID = req.RelatedContactID
	if err := s.db.Save(&relationship).Error; err != nil {
		return nil, err
	}
	resp := toRelationshipResponse(&relationship)
	return &resp, nil
}

func (s *RelationshipService) Delete(id uint, contactID, vaultID string) error {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return err
	}

	var relationship models.Relationship
	if err := s.db.Where("id = ? AND contact_id = ?", id, contactID).First(&relationship).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRelationshipNotFound
		}
		return err
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&relationship).Error; err != nil {
			return err
		}

		reverseTypeID, found := findReverseTypeID(tx, relationship.RelationshipTypeID)
		if found {
			// Tolerant delete — ignore if reverse doesn't exist
			tx.Where("contact_id = ? AND related_contact_id = ? AND relationship_type_id = ?",
				relationship.RelatedContactID, relationship.ContactID, reverseTypeID).
				Delete(&models.Relationship{})
		}

		return nil
	})
}

func findReverseTypeID(tx *gorm.DB, typeID uint) (uint, bool) {
	var rt models.RelationshipType
	if err := tx.First(&rt, typeID).Error; err != nil {
		return 0, false
	}
	if rt.Name == nil || rt.NameReverseRelationship == nil {
		return 0, false
	}
	if *rt.Name == *rt.NameReverseRelationship {
		return rt.ID, true
	}
	var reverseType models.RelationshipType
	err := tx.Where("relationship_group_type_id = ? AND name = ?", rt.RelationshipGroupTypeID, *rt.NameReverseRelationship).
		First(&reverseType).Error
	if err != nil {
		return 0, false
	}
	return reverseType.ID, true
}

func toRelationshipResponse(r *models.Relationship) dto.RelationshipResponse {
	typeName := ""
	if r.RelationshipType.Name != nil {
		typeName = *r.RelationshipType.Name
	}
	return dto.RelationshipResponse{
		ID:                   r.ID,
		ContactID:            r.ContactID,
		RelatedContactID:     r.RelatedContactID,
		RelationshipTypeID:   r.RelationshipTypeID,
		RelationshipTypeName: typeName,
		CreatedAt:            r.CreatedAt,
		UpdatedAt:            r.UpdatedAt,
	}
}

func contactLabel(c *models.Contact) string {
	var parts []string
	if c.FirstName != nil && *c.FirstName != "" {
		parts = append(parts, *c.FirstName)
	}
	if c.LastName != nil && *c.LastName != "" {
		parts = append(parts, *c.LastName)
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func (s *RelationshipService) GetContactGraph(contactID, vaultID string) (*dto.ContactGraphResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}

	var center models.Contact
	if err := s.db.Where("id = ?", contactID).First(&center).Error; err != nil {
		return nil, ErrContactNotFound
	}

	var relationships []models.Relationship
	if err := s.db.
		Preload("RelationshipType").
		Preload("Contact").
		Preload("RelatedContact").
		Where("contact_id = ? OR related_contact_id = ?", contactID, contactID).
		Find(&relationships).Error; err != nil {
		return nil, err
	}

	nodeMap := make(map[string]dto.GraphNode)
	nodeMap[contactID] = dto.GraphNode{
		ID:       center.ID,
		Label:    contactLabel(&center),
		IsCenter: true,
	}

	var edges []dto.GraphEdge
	connectedIDs := make(map[string]bool)

	for _, r := range relationships {
		if _, exists := nodeMap[r.Contact.ID]; !exists {
			nodeMap[r.Contact.ID] = dto.GraphNode{
				ID:    r.Contact.ID,
				Label: contactLabel(&r.Contact),
			}
		}
		if _, exists := nodeMap[r.RelatedContact.ID]; !exists {
			nodeMap[r.RelatedContact.ID] = dto.GraphNode{
				ID:    r.RelatedContact.ID,
				Label: contactLabel(&r.RelatedContact),
			}
		}

		if r.ContactID != contactID {
			connectedIDs[r.ContactID] = true
		}
		if r.RelatedContactID != contactID {
			connectedIDs[r.RelatedContactID] = true
		}

		typeName := ""
		if r.RelationshipType.Name != nil {
			typeName = *r.RelationshipType.Name
		}
		edges = append(edges, dto.GraphEdge{
			Source: r.ContactID,
			Target: r.RelatedContactID,
			Type:   typeName,
		})
	}

	if len(connectedIDs) > 0 {
		ids := make([]string, 0, len(connectedIDs))
		for id := range connectedIDs {
			ids = append(ids, id)
		}

		var secondLayerRels []models.Relationship
		if err := s.db.
			Preload("RelationshipType").
			Where("contact_id IN ? AND related_contact_id IN ?", ids, ids).
			Find(&secondLayerRels).Error; err != nil {
			return nil, err
		}

		for _, r := range secondLayerRels {
			typeName := ""
			if r.RelationshipType.Name != nil {
				typeName = *r.RelationshipType.Name
			}
			edges = append(edges, dto.GraphEdge{
				Source: r.ContactID,
				Target: r.RelatedContactID,
				Type:   typeName,
			})
		}
	}

	nodes := make([]dto.GraphNode, 0, len(nodeMap))
	for _, n := range nodeMap {
		nodes = append(nodes, n)
	}

	return &dto.ContactGraphResponse{
		Nodes: nodes,
		Edges: edges,
	}, nil
}

func (s *RelationshipService) CalculateKinship(contactID1, contactID2, vaultID string) (*dto.KinshipResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID1, vaultID); err != nil {
		return nil, err
	}
	if err := validateContactBelongsToVault(s.db, contactID2, vaultID); err != nil {
		return nil, err
	}

	var relationships []models.Relationship
	if err := s.db.
		Preload("RelationshipType").
		Where("contact_id IN (SELECT id FROM contacts WHERE vault_id = ?)", vaultID).
		Find(&relationships).Error; err != nil {
		return nil, err
	}

	type edge struct {
		to     string
		weight int
	}
	adj := make(map[string][]edge)

	for _, r := range relationships {
		if r.RelationshipType.Degree == nil {
			continue
		}
		w := *r.RelationshipType.Degree
		adj[r.ContactID] = append(adj[r.ContactID], edge{to: r.RelatedContactID, weight: w})
		adj[r.RelatedContactID] = append(adj[r.RelatedContactID], edge{to: r.ContactID, weight: w})
	}

	dist := make(map[string]int)
	prev := make(map[string]string)
	dist[contactID1] = 0

	pq := &priorityQueue{}
	heap.Init(pq)
	heap.Push(pq, &pqItem{id: contactID1, dist: 0})

	for pq.Len() > 0 {
		cur := heap.Pop(pq).(*pqItem)
		if cur.dist > getDist(dist, cur.id) {
			continue
		}
		if cur.id == contactID2 {
			break
		}
		for _, e := range adj[cur.id] {
			newDist := cur.dist + e.weight
			if newDist < getDist(dist, e.to) {
				dist[e.to] = newDist
				prev[e.to] = cur.id
				heap.Push(pq, &pqItem{id: e.to, dist: newDist})
			}
		}
	}

	if _, ok := dist[contactID2]; !ok {
		return &dto.KinshipResponse{Degree: nil, Path: nil}, nil
	}

	var path []string
	for cur := contactID2; cur != ""; cur = prev[cur] {
		path = append([]string{cur}, path...)
		if cur == contactID1 {
			break
		}
	}

	degree := dist[contactID2]
	return &dto.KinshipResponse{
		Degree: &degree,
		Path:   path,
	}, nil
}

func getDist(dist map[string]int, id string) int {
	if d, ok := dist[id]; ok {
		return d
	}
	return math.MaxInt64
}

type pqItem struct {
	id    string
	dist  int
	index int
}

type priorityQueue []*pqItem

func (pq priorityQueue) Len() int           { return len(pq) }
func (pq priorityQueue) Less(i, j int) bool { return pq[i].dist < pq[j].dist }
func (pq priorityQueue) Swap(i, j int)      { pq[i], pq[j] = pq[j], pq[i]; pq[i].index = i; pq[j].index = j }
func (pq *priorityQueue) Push(x interface{}) {
	item := x.(*pqItem)
	item.index = len(*pq)
	*pq = append(*pq, item)
}
func (pq *priorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*pq = old[:n-1]
	return item
}
