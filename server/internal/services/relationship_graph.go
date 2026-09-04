package services

import (
	"sort"
	"strconv"
	"strings"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/i18n"
	"github.com/naiba/bonds/internal/models"
)

const (
	relKindCustom      = "custom"
	relKindParent      = "parent"
	relKindChild       = "child"
	relKindSibling     = "sibling"
	relKindSpouse      = "spouse"
	relKindGrandParent = "grand_parent"
	relKindGrandChild  = "grand_child"
	relKindAncestor    = "ancestor"
	relKindDescendant  = "descendant"
	relKindStepParent  = "step_parent"
	relKindStepChild   = "step_child"
	relKindStepSibling = "step_sibling"
	relKindParentInLaw = "parent_in_law"
	relKindChildInLaw  = "child_in_law"
	relKindUncleAunt   = "uncle_aunt"
	relKindNephewNiece = "nephew_niece"
	relKindCousin      = "cousin"
	relKeyParent       = "seed.relationship_types.parent"
	relKeyChild        = "seed.relationship_types.child"
	relKeySibling      = "seed.relationship_types.brother_sister"
	relKeySpouse       = "seed.relationship_types.spouse"
	relKeyGrandParent  = "seed.relationship_types.grand_parent"
	relKeyGrandChild   = "seed.relationship_types.grand_child"
	relKeyUncleAunt    = "seed.relationship_types.uncle_aunt"
	relKeyNephewNiece  = "seed.relationship_types.nephew_niece"
	relKeyCousin       = "seed.relationship_types.cousin"
)

type contactSet map[string]struct{}

type contactPair struct {
	first  string
	second string
}

type graphEdgeBuilder struct {
	source    string
	target    string
	relations map[string]dto.GraphRelation
}

type relationshipGraphBuilder struct {
	edges              map[contactPair]*graphEdgeBuilder
	directRelationKind map[contactPair]map[string]struct{}
	parentsByChild     map[string]contactSet
	childrenByParent   map[string]contactSet
	spouses            map[string]contactSet
	bloodSiblings      map[string]contactSet
	locale             string
}

func newRelationshipGraphBuilder(locale string) *relationshipGraphBuilder {
	return &relationshipGraphBuilder{
		edges:              make(map[contactPair]*graphEdgeBuilder),
		directRelationKind: make(map[contactPair]map[string]struct{}),
		parentsByChild:     make(map[string]contactSet),
		childrenByParent:   make(map[string]contactSet),
		spouses:            make(map[string]contactSet),
		bloodSiblings:      make(map[string]contactSet),
		locale:             locale,
	}
}

// GetContactGraph returns the complete explicit connected component rooted at
// contactID and augments it with read-only inferred family relationships. It
// never persists inferred rows. Reciprocal database rows are folded into one
// edge with labels for both endpoint perspectives.
func (s *RelationshipService) GetContactGraph(contactID, vaultID, userID, locale string) (*dto.ContactGraphResponse, error) {
	if err := validateContactBelongsToVault(s.db, contactID, vaultID); err != nil {
		return nil, err
	}

	var center models.Contact
	if err := s.db.Where("id = ?", contactID).First(&center).Error; err != nil {
		return nil, ErrContactNotFound
	}

	accessibleVaults, err := accessibleVaultIDSet(s.db, userID)
	if err != nil {
		return nil, err
	}
	accessibleVaultIDs := vaultIDList(accessibleVaults)

	var relationships []models.Relationship
	if len(accessibleVaultIDs) > 0 {
		if err := s.db.
			Preload("RelationshipType").
			Preload("Contact").
			Preload("RelatedContact").
			Where("contact_id IN (SELECT id FROM contacts WHERE vault_id IN ?)", accessibleVaultIDs).
			Order("id ASC").
			Find(&relationships).Error; err != nil {
			return nil, err
		}
	}

	// Build an undirected explicit adjacency list first. Inferred relationships
	// cannot introduce new nodes, so a BFS over this list yields the complete
	// component while preserving cross-vault read isolation.
	adjacency := make(map[string]contactSet)
	contacts := map[string]models.Contact{contactID: center}
	readableRelationships := make([]models.Relationship, 0, len(relationships))
	for _, relationship := range relationships {
		if !canReadContactInVault(accessibleVaults, relationship.Contact) ||
			!canReadContactInVault(accessibleVaults, relationship.RelatedContact) {
			continue
		}
		if relationship.ContactID == relationship.RelatedContactID {
			continue
		}
		readableRelationships = append(readableRelationships, relationship)
		contacts[relationship.ContactID] = relationship.Contact
		contacts[relationship.RelatedContactID] = relationship.RelatedContact
		addToContactMap(adjacency, relationship.ContactID, relationship.RelatedContactID)
		addToContactMap(adjacency, relationship.RelatedContactID, relationship.ContactID)
	}

	reachable := contactSet{contactID: {}}
	queue := []string{contactID}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for neighbor := range adjacency[current] {
			if _, seen := reachable[neighbor]; seen {
				continue
			}
			reachable[neighbor] = struct{}{}
			queue = append(queue, neighbor)
		}
	}

	builder := newRelationshipGraphBuilder(locale)
	for _, relationship := range readableRelationships {
		if _, ok := reachable[relationship.ContactID]; !ok {
			continue
		}
		if _, ok := reachable[relationship.RelatedContactID]; !ok {
			continue
		}
		builder.addExplicitRelationship(relationship)
	}
	builder.inferFamilyRelationships(reachable)

	formatter, err := newContactNameFormatter(s.db, userID)
	if err != nil {
		return nil, err
	}
	nodes := make([]dto.GraphNode, 0, len(reachable))
	for id := range reachable {
		contact, ok := contacts[id]
		if !ok {
			continue
		}
		label, err := formatter.format(&contact, "")
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, dto.GraphNode{ID: id, Label: label, IsCenter: id == contactID})
	}
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].IsCenter != nodes[j].IsCenter {
			return nodes[i].IsCenter
		}
		if nodes[i].Label != nodes[j].Label {
			return nodes[i].Label < nodes[j].Label
		}
		return nodes[i].ID < nodes[j].ID
	})

	return &dto.ContactGraphResponse{Nodes: nodes, Edges: builder.buildEdges()}, nil
}

func (b *relationshipGraphBuilder) addExplicitRelationship(relationship models.Relationship) {
	sourceKind := graphKindForTranslationKey(stringValue(relationship.RelationshipType.NameTranslationKey))
	targetKind := graphKindForTranslationKey(stringValue(relationship.RelationshipType.NameReverseRelationshipTranslationKey))
	sourceLabel := stringValue(relationship.RelationshipType.Name)
	targetLabel := stringValue(relationship.RelationshipType.NameReverseRelationship)
	if targetLabel == "" {
		targetLabel = sourceLabel
	}
	b.addRelation(
		relationship.ContactID,
		relationship.RelatedContactID,
		sourceKind,
		targetKind,
		sourceLabel,
		targetLabel,
		false,
		0,
	)

	switch sourceKind {
	case relKindParent:
		b.addParent(relationship.ContactID, relationship.RelatedContactID)
	case relKindChild:
		b.addParent(relationship.RelatedContactID, relationship.ContactID)
	case relKindSpouse:
		b.addSpouse(relationship.ContactID, relationship.RelatedContactID)
	case relKindSibling:
		b.addBloodSibling(relationship.ContactID, relationship.RelatedContactID)
	}
}

func (b *relationshipGraphBuilder) inferFamilyRelationships(reachable contactSet) {
	// Contacts sharing one or more direct parents are siblings. A set keeps
	// half/full siblings and multiple shared parents from producing duplicates.
	for _, children := range b.childrenByParent {
		childIDs := sortedContactIDs(children)
		for i := 0; i < len(childIDs); i++ {
			for j := i + 1; j < len(childIDs); j++ {
				b.addBloodSibling(childIDs[i], childIDs[j])
			}
		}
	}
	for left, siblings := range b.bloodSiblings {
		for right := range siblings {
			if left >= right {
				continue
			}
			b.addInferredRelation(
				left, right,
				relKindSibling, relKindSibling,
				i18n.T(b.locale, relKeySibling), i18n.T(b.locale, relKeySibling),
				0,
			)
		}
	}

	// Compute the transitive parent closure independently for every descendant.
	// The shortest generation distance wins when malformed data offers multiple
	// paths, and visited prevents parent cycles from creating self relations.
	for descendant := range reachable {
		distance := map[string]int{descendant: 0}
		queue := []string{descendant}
		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]
			for parent := range b.parentsByChild[current] {
				if _, ok := reachable[parent]; !ok {
					continue
				}
				if _, seen := distance[parent]; seen {
					continue
				}
				distance[parent] = distance[current] + 1
				queue = append(queue, parent)
			}
		}
		for ancestor, generations := range distance {
			if ancestor == descendant || generations < 2 {
				continue
			}
			sourceKind, targetKind := relKindAncestor, relKindDescendant
			sourceLabel := i18n.Tt(b.locale, "graph.relationships.ancestor_generations", map[string]string{"count": strconv.Itoa(generations)})
			targetLabel := i18n.Tt(b.locale, "graph.relationships.descendant_generations", map[string]string{"count": strconv.Itoa(generations)})
			if generations == 2 {
				sourceKind, targetKind = relKindGrandParent, relKindGrandChild
				sourceLabel = i18n.T(b.locale, relKeyGrandParent)
				targetLabel = i18n.T(b.locale, relKeyGrandChild)
			}
			b.addInferredRelation(ancestor, descendant, sourceKind, targetKind, sourceLabel, targetLabel, generations)
		}
	}

	// A parent's spouse is a step-parent only when that spouse is not already a
	// recorded direct parent. This is what lets biological parents and
	// step-parents coexist without misclassifying either one.
	for parent, spouses := range b.spouses {
		for spouse := range spouses {
			for child := range b.childrenByParent[parent] {
				if child == spouse || containsContact(b.parentsByChild[child], spouse) {
					continue
				}
				b.addInferredRelation(
					spouse, child,
					relKindStepParent, relKindStepChild,
					i18n.T(b.locale, "graph.relationships.step_parent"),
					i18n.T(b.locale, "graph.relationships.step_child"),
					1,
				)
			}
		}
	}

	// A child's spouse is the parent's child-in-law, not their step-child. This
	// is deliberately separate from the rule above: the spouse of a parent is a
	// step-parent, while the spouse of a child is an in-law.
	for parent, children := range b.childrenByParent {
		for child := range children {
			for spouse := range b.spouses[child] {
				if spouse == parent {
					continue
				}
				b.addInferredRelation(
					parent, spouse,
					relKindParentInLaw, relKindChildInLaw,
					i18n.T(b.locale, "graph.relationships.parent_in_law"),
					i18n.T(b.locale, "graph.relationships.child_in_law"),
					1,
				)
			}
		}
	}

	// Children brought into a spousal relationship by different parents are
	// step-siblings, unless they already share a parent (full/half siblings).
	seenSpouses := make(map[contactPair]struct{})
	for left, spouses := range b.spouses {
		for right := range spouses {
			pair := makeContactPair(left, right)
			if _, seen := seenSpouses[pair]; seen {
				continue
			}
			seenSpouses[pair] = struct{}{}
			for leftChild := range b.childrenByParent[left] {
				for rightChild := range b.childrenByParent[right] {
					if leftChild == rightChild || shareContact(b.parentsByChild[leftChild], b.parentsByChild[rightChild]) {
						continue
					}
					b.addInferredRelation(
						leftChild, rightChild,
						relKindStepSibling, relKindStepSibling,
						i18n.T(b.locale, "graph.relationships.step_sibling"),
						i18n.T(b.locale, "graph.relationships.step_sibling"),
						0,
					)
				}
			}
		}
	}

	// Siblings of a direct parent are uncles/aunts. Their direct children are
	// cousins. Blood siblings and explicitly recorded sibling relationships both
	// participate, while step-siblings deliberately do not.
	for child, parents := range b.parentsByChild {
		for parent := range parents {
			for parentSibling := range b.bloodSiblings[parent] {
				if parentSibling == child || containsContact(parents, parentSibling) {
					continue
				}
				b.addInferredRelation(
					parentSibling, child,
					relKindUncleAunt, relKindNephewNiece,
					i18n.T(b.locale, relKeyUncleAunt),
					i18n.T(b.locale, relKeyNephewNiece),
					0,
				)
				for cousin := range b.childrenByParent[parentSibling] {
					if cousin == child || shareContact(parents, b.parentsByChild[cousin]) {
						continue
					}
					b.addInferredRelation(
						child, cousin,
						relKindCousin, relKindCousin,
						i18n.T(b.locale, relKeyCousin), i18n.T(b.locale, relKeyCousin),
						0,
					)
				}
			}
		}
	}
}

func (b *relationshipGraphBuilder) addParent(parent, child string) {
	if parent == child {
		return
	}
	addToContactMap(b.parentsByChild, child, parent)
	addToContactMap(b.childrenByParent, parent, child)
}

func (b *relationshipGraphBuilder) addSpouse(left, right string) {
	if left == right {
		return
	}
	addToContactMap(b.spouses, left, right)
	addToContactMap(b.spouses, right, left)
}

func (b *relationshipGraphBuilder) addBloodSibling(left, right string) {
	if left == right {
		return
	}
	addToContactMap(b.bloodSiblings, left, right)
	addToContactMap(b.bloodSiblings, right, left)
}

func (b *relationshipGraphBuilder) addInferredRelation(source, target, sourceKind, targetKind, sourceLabel, targetLabel string, generations int) {
	pair, swapped := canonicalPair(source, target)
	canonicalSourceKind, canonicalTargetKind := sourceKind, targetKind
	if swapped {
		canonicalSourceKind, canonicalTargetKind = targetKind, sourceKind
	}
	if kinds := b.directRelationKind[pair]; kinds != nil {
		if _, exists := kinds[relationKindKey(canonicalSourceKind, canonicalTargetKind)]; exists {
			return
		}
	}
	b.addRelation(source, target, sourceKind, targetKind, sourceLabel, targetLabel, true, generations)
}

func (b *relationshipGraphBuilder) addRelation(source, target, sourceKind, targetKind, sourceLabel, targetLabel string, inferred bool, generations int) {
	if source == target {
		return
	}
	pair, swapped := canonicalPair(source, target)
	if swapped {
		sourceKind, targetKind = targetKind, sourceKind
		sourceLabel, targetLabel = targetLabel, sourceLabel
	}
	edge := b.edges[pair]
	if edge == nil {
		edge = &graphEdgeBuilder{source: pair.first, target: pair.second, relations: make(map[string]dto.GraphRelation)}
		b.edges[pair] = edge
	}
	relation := dto.GraphRelation{
		SourceKind: sourceKind, TargetKind: targetKind,
		SourceLabel: sourceLabel, TargetLabel: targetLabel,
		Inferred: inferred, Generations: generations,
	}
	key := graphRelationKey(relation)
	edge.relations[key] = relation
	if !inferred {
		if b.directRelationKind[pair] == nil {
			b.directRelationKind[pair] = make(map[string]struct{})
		}
		b.directRelationKind[pair][relationKindKey(sourceKind, targetKind)] = struct{}{}
	}
}

func (b *relationshipGraphBuilder) buildEdges() []dto.GraphEdge {
	pairs := make([]contactPair, 0, len(b.edges))
	for pair := range b.edges {
		pairs = append(pairs, pair)
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].first != pairs[j].first {
			return pairs[i].first < pairs[j].first
		}
		return pairs[i].second < pairs[j].second
	})

	edges := make([]dto.GraphEdge, 0, len(pairs))
	for _, pair := range pairs {
		builder := b.edges[pair]
		relations := make([]dto.GraphRelation, 0, len(builder.relations))
		allInferred := true
		for _, relation := range builder.relations {
			relations = append(relations, relation)
			if !relation.Inferred {
				allInferred = false
			}
		}
		sort.Slice(relations, func(i, j int) bool {
			if relations[i].Inferred != relations[j].Inferred {
				return !relations[i].Inferred
			}
			if relations[i].SourceLabel != relations[j].SourceLabel {
				return relations[i].SourceLabel < relations[j].SourceLabel
			}
			return relations[i].TargetLabel < relations[j].TargetLabel
		})
		typeLabels := make([]string, 0, len(relations))
		for _, relation := range relations {
			if relation.SourceLabel != "" {
				typeLabels = append(typeLabels, relation.SourceLabel)
			}
		}
		edges = append(edges, dto.GraphEdge{
			Source: builder.source, Target: builder.target,
			Type: strings.Join(typeLabels, " · "), Inferred: allInferred,
			Relations: relations,
		})
	}
	return edges
}

func graphKindForTranslationKey(key string) string {
	switch key {
	case relKeyParent:
		return relKindParent
	case relKeyChild:
		return relKindChild
	case relKeySibling:
		return relKindSibling
	case relKeySpouse:
		return relKindSpouse
	case relKeyGrandParent:
		return relKindGrandParent
	case relKeyGrandChild:
		return relKindGrandChild
	case relKeyUncleAunt:
		return relKindUncleAunt
	case relKeyNephewNiece:
		return relKindNephewNiece
	case relKeyCousin:
		return relKindCousin
	default:
		return relKindCustom
	}
}

func canonicalPair(left, right string) (contactPair, bool) {
	if left <= right {
		return contactPair{first: left, second: right}, false
	}
	return contactPair{first: right, second: left}, true
}

func makeContactPair(left, right string) contactPair {
	pair, _ := canonicalPair(left, right)
	return pair
}

func addToContactMap(values map[string]contactSet, key, value string) {
	if values[key] == nil {
		values[key] = make(contactSet)
	}
	values[key][value] = struct{}{}
}

func containsContact(values contactSet, contactID string) bool {
	_, ok := values[contactID]
	return ok
}

func shareContact(left, right contactSet) bool {
	if len(left) > len(right) {
		left, right = right, left
	}
	for value := range left {
		if containsContact(right, value) {
			return true
		}
	}
	return false
}

func sortedContactIDs(values contactSet) []string {
	ids := make([]string, 0, len(values))
	for id := range values {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func relationKindKey(sourceKind, targetKind string) string {
	return sourceKind + "\x00" + targetKind
}

func graphRelationKey(relation dto.GraphRelation) string {
	return strings.Join([]string{
		relationKindKey(relation.SourceKind, relation.TargetKind),
		relation.SourceLabel,
		relation.TargetLabel,
		strconv.FormatBool(relation.Inferred),
		strconv.Itoa(relation.Generations),
	}, "\x00")
}

// The node limit is a target budget rather than a hard cap: the largest
// connected component is kept whole even when it exceeds the requested value.
// Clamp user input to the largest supported UI option so a hand-written URL
// cannot request an arbitrarily large collection of otherwise-droppable
// components.
const (
	defaultVaultGraphNodeLimit = 1000
	maxVaultGraphNodeLimit     = 5000
)

// GetVaultGraph returns the relationship graph of one vault: every contact in
// it that is related to another contact in it, and the relationships between
// them, with the same inference the single-contact graph applies.
//
// Three deliberate exclusions, each reported back so the page can say what is
// missing rather than quietly showing less than the whole truth:
//
//   - Contacts with no in-vault relationship. A vault is mostly people who are
//     in it for their own sake, and drawing them as unconnected dots buries the
//     structure the page exists to show.
//   - Relationships that leave the vault. bonds allows them, but a vault-scoped
//     view that silently pulled in contacts from elsewhere would be a different
//     vault's business rendered on this vault's page.
//   - Whole components past the node limit, largest first, so the cap never
//     cuts a family in half.
func (s *RelationshipService) GetVaultGraph(vaultID, userID, locale string, limit int, filter VaultGraphFilter) (*dto.VaultGraphResponse, error) {
	limit = normalizeVaultGraphNodeLimit(limit)

	accessibleVaults, err := accessibleVaultIDSet(s.db, userID)
	if err != nil {
		return nil, err
	}
	if _, ok := accessibleVaults[vaultID]; !ok {
		return nil, ErrVaultNotFound
	}

	var contacts []models.Contact
	if err := s.db.Where("vault_id = ?", vaultID).Find(&contacts).Error; err != nil {
		return nil, err
	}
	contactsByID := make(map[string]models.Contact, len(contacts))
	for _, contact := range contacts {
		contactsByID[contact.ID] = contact
	}

	var relationships []models.Relationship
	if err := s.db.
		Preload("RelationshipType").
		Preload("Contact").
		Preload("RelatedContact").
		Where("contact_id IN (SELECT id FROM contacts WHERE vault_id = ?)", vaultID).
		Order("id ASC").
		Find(&relationships).Error; err != nil {
		return nil, err
	}

	// Split the relationships into the ones that stay inside the vault and the
	// ones that leave it. The second group is counted, not drawn.
	adjacency := make(map[string]contactSet)
	internal := make([]models.Relationship, 0, len(relationships))
	external := 0
	for _, relationship := range relationships {
		if relationship.ContactID == relationship.RelatedContactID {
			continue
		}
		_, sourceInVault := contactsByID[relationship.ContactID]
		_, targetInVault := contactsByID[relationship.RelatedContactID]
		if !sourceInVault || !targetInVault {
			// Only count it if the far end is one this user could otherwise see;
			// a relationship into a vault they have no access to is not
			// something to advertise the existence of.
			if canReadContactInVault(accessibleVaults, relationship.RelatedContact) {
				external++
			}
			continue
		}
		internal = append(internal, relationship)
		addToContactMap(adjacency, relationship.ContactID, relationship.RelatedContactID)
		addToContactMap(adjacency, relationship.RelatedContactID, relationship.ContactID)
	}

	// The facet options describe the unfiltered graph, so the reader can always
	// see the values they did not pick. The filter is applied after them.
	facets, err := s.loadContactFacets(vaultID, locale, contacts)
	if err != nil {
		return nil, err
	}
	drawable := contactSet{}
	for id := range adjacency {
		drawable[id] = struct{}{}
	}

	filtered, filteredAdjacency := applyGraphFilter(adjacency, facets, filter)

	components := connectedComponents(filteredAdjacency)
	kept, keptComponents := componentsWithinLimit(components, limit)

	builder := newRelationshipGraphBuilder(locale)
	for _, relationship := range internal {
		if !containsContact(kept, relationship.ContactID) {
			continue
		}
		if !containsContact(kept, relationship.RelatedContactID) {
			continue
		}
		builder.addExplicitRelationship(relationship)
	}
	builder.inferFamilyRelationships(kept)

	formatter, err := newContactNameFormatter(s.db, userID)
	if err != nil {
		return nil, err
	}
	nodes := make([]dto.GraphNode, 0, len(kept))
	for id := range kept {
		contact, ok := contactsByID[id]
		if !ok {
			continue
		}
		label, err := formatter.format(&contact, "")
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, dto.GraphNode{ID: id, Label: label})
	}
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].Label != nodes[j].Label {
			return nodes[i].Label < nodes[j].Label
		}
		return nodes[i].ID < nodes[j].ID
	})

	return &dto.VaultGraphResponse{
		Nodes:                 nodes,
		Edges:                 builder.buildEdges(),
		Components:            keptComponents,
		IsolatedContacts:      len(contactsByID) - len(adjacency),
		ExternalRelationships: external,
		Truncated:             keptComponents < len(components),
		Facets:                facets.options(drawable),
		FilteredOut:           len(drawable) - len(filtered),
	}, nil
}

func normalizeVaultGraphNodeLimit(limit int) int {
	if limit <= 0 {
		return defaultVaultGraphNodeLimit
	}
	if limit > maxVaultGraphNodeLimit {
		return maxVaultGraphNodeLimit
	}
	return limit
}

// applyGraphFilter narrows an adjacency list to the contacts matching the
// filter, then drops any of those left with no partner still standing.
//
// The second step matters: a contact who matches but whose every relation was
// filtered away has nothing to show on a relationship graph, and drawing them
// as a loose dot is the same noise isolated contacts were excluded to avoid.
// Both losses are reported together, because to the reader they are one thing —
// people the filter took off the canvas.
func applyGraphFilter(adjacency map[string]contactSet, facets *contactFacets, filter VaultGraphFilter) (contactSet, map[string]contactSet) {
	if len(filter) == 0 {
		matched := contactSet{}
		for id := range adjacency {
			matched[id] = struct{}{}
		}
		return matched, adjacency
	}

	matched := contactSet{}
	for id := range adjacency {
		if facets.matches(id, filter) {
			matched[id] = struct{}{}
		}
	}

	filteredAdjacency := make(map[string]contactSet, len(matched))
	for id := range matched {
		neighbours := contactSet{}
		for neighbour := range adjacency[id] {
			if _, ok := matched[neighbour]; ok {
				neighbours[neighbour] = struct{}{}
			}
		}
		if len(neighbours) == 0 {
			continue
		}
		filteredAdjacency[id] = neighbours
	}

	drawn := contactSet{}
	for id := range filteredAdjacency {
		drawn[id] = struct{}{}
	}
	return drawn, filteredAdjacency
}

// connectedComponents groups an undirected adjacency list into its connected
// components, largest first.
func connectedComponents(adjacency map[string]contactSet) []contactSet {
	seen := contactSet{}
	components := make([]contactSet, 0)
	for _, start := range sortedAdjacencyKeys(adjacency) {
		if _, done := seen[start]; done {
			continue
		}
		component := contactSet{start: {}}
		seen[start] = struct{}{}
		queue := []string{start}
		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]
			for _, neighbor := range sortedContactIDs(adjacency[current]) {
				if _, done := seen[neighbor]; done {
					continue
				}
				seen[neighbor] = struct{}{}
				component[neighbor] = struct{}{}
				queue = append(queue, neighbor)
			}
		}
		components = append(components, component)
	}
	sort.SliceStable(components, func(i, j int) bool {
		if len(components[i]) != len(components[j]) {
			return len(components[i]) > len(components[j])
		}
		return sortedContactIDs(components[i])[0] < sortedContactIDs(components[j])[0]
	})
	return components
}

// componentsWithinLimit takes whole components, largest first, and stops at the
// first one that would not fit. It returns the union of what it took and how
// many components that was. The largest component is always taken even when it
// exceeds the limit on its own, because returning an empty graph is a worse
// answer than returning one oversized cluster the caller can be told about.
func componentsWithinLimit(components []contactSet, limit int) (contactSet, int) {
	kept := contactSet{}
	for index, component := range components {
		if index > 0 && len(kept)+len(component) > limit {
			// Stop rather than skip: components arrive largest first, so a
			// later smaller one squeezing in would drop a bigger cluster in
			// favour of a lesser one.
			return kept, index
		}
		for id := range component {
			kept[id] = struct{}{}
		}
	}
	return kept, len(components)
}

func sortedAdjacencyKeys(adjacency map[string]contactSet) []string {
	keys := make([]string, 0, len(adjacency))
	for key := range adjacency {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
