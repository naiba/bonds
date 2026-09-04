package services

import (
	"testing"

	"github.com/naiba/bonds/internal/dto"
	"github.com/naiba/bonds/internal/models"
)

func createGraphContact(t *testing.T, ctx relationshipTestCtx, firstName string) string {
	t.Helper()
	contact, err := NewContactService(ctx.db).CreateContact(
		ctx.vaultID,
		ctx.userID,
		dto.CreateContactRequest{FirstName: firstName},
	)
	if err != nil {
		t.Fatalf("create graph contact %q: %v", firstName, err)
	}
	return contact.ID
}

func seededGraphRelationshipTypeID(t *testing.T, ctx relationshipTestCtx, translationKey string) uint {
	t.Helper()
	var relationshipType models.RelationshipType
	if err := ctx.db.
		Joins("JOIN relationship_group_types ON relationship_group_types.id = relationship_types.relationship_group_type_id").
		Where("relationship_group_types.account_id = ? AND relationship_types.name_translation_key = ?", ctx.accountID, translationKey).
		First(&relationshipType).Error; err != nil {
		t.Fatalf("find relationship type %q: %v", translationKey, err)
	}
	return relationshipType.ID
}

func createGraphRelationship(t *testing.T, ctx relationshipTestCtx, sourceID, targetID string, relationshipTypeID uint) {
	t.Helper()
	if _, err := ctx.svc.Create(sourceID, ctx.vaultID, ctx.userID, dto.CreateRelationshipRequest{
		RelationshipTypeID: relationshipTypeID,
		RelatedContactID:   targetID,
	}); err != nil {
		t.Fatalf("create graph relationship %s -> %s: %v", sourceID, targetID, err)
	}
}

func graphContainsNode(graph *dto.ContactGraphResponse, contactID string) bool {
	for _, node := range graph.Nodes {
		if node.ID == contactID {
			return true
		}
	}
	return false
}

func graphRelationCount(graph *dto.ContactGraphResponse, leftID, rightID, leftKind, rightKind string, inferred bool) int {
	count := 0
	for _, edge := range graph.Edges {
		if !((edge.Source == leftID && edge.Target == rightID) || (edge.Source == rightID && edge.Target == leftID)) {
			continue
		}
		for _, relation := range edge.Relations {
			relationLeftKind, relationRightKind := relation.SourceKind, relation.TargetKind
			if edge.Source != leftID {
				relationLeftKind, relationRightKind = relation.TargetKind, relation.SourceKind
			}
			if relationLeftKind == leftKind && relationRightKind == rightKind && relation.Inferred == inferred {
				count++
			}
		}
	}
	return count
}

func graphRelationGenerations(graph *dto.ContactGraphResponse, leftID, rightID, leftKind, rightKind string) int {
	for _, edge := range graph.Edges {
		if !((edge.Source == leftID && edge.Target == rightID) || (edge.Source == rightID && edge.Target == leftID)) {
			continue
		}
		for _, relation := range edge.Relations {
			relationLeftKind, relationRightKind := relation.SourceKind, relation.TargetKind
			if edge.Source != leftID {
				relationLeftKind, relationRightKind = relation.TargetKind, relation.SourceKind
			}
			if relationLeftKind == leftKind && relationRightKind == rightKind {
				return relation.Generations
			}
		}
	}
	return 0
}

func TestContactGraphTraversesCompleteComponentAndCollapsesReciprocalRows(t *testing.T) {
	ctx := setupRelationshipTestFull(t)
	third := createGraphContact(t, ctx, "Third")
	fourth := createGraphContact(t, ctx, "Fourth")
	isolated := createGraphContact(t, ctx, "Isolated")
	forwardTypeID, _ := createAsymmetricTypePair(t, ctx.db, ctx.accountID)

	createGraphRelationship(t, ctx, ctx.contactID, ctx.relatedContactID, forwardTypeID)
	createGraphRelationship(t, ctx, ctx.relatedContactID, third, forwardTypeID)
	createGraphRelationship(t, ctx, third, fourth, forwardTypeID)

	graph, err := ctx.svc.GetContactGraph(ctx.contactID, ctx.vaultID, ctx.userID, "en")
	if err != nil {
		t.Fatalf("GetContactGraph: %v", err)
	}
	for _, contactID := range []string{ctx.contactID, ctx.relatedContactID, third, fourth} {
		if !graphContainsNode(graph, contactID) {
			t.Errorf("complete component missing contact %s", contactID)
		}
	}
	if graphContainsNode(graph, isolated) {
		t.Fatal("graph included an isolated contact")
	}
	if len(graph.Nodes) != 4 {
		t.Fatalf("nodes = %d, want 4", len(graph.Nodes))
	}
	if len(graph.Edges) != 3 {
		t.Fatalf("edges = %d, want one edge per reciprocal pair (3)", len(graph.Edges))
	}
	for _, edge := range graph.Edges {
		if edge.Inferred {
			t.Errorf("explicit edge marked inferred: %#v", edge)
		}
		if len(edge.Relations) != 1 {
			t.Errorf("reciprocal rows were not collapsed: %#v", edge)
		}
	}
}

func TestContactGraphInfersStepFamilyWithoutMisclassifyingRecordedParents(t *testing.T) {
	ctx := setupRelationshipTestFull(t)
	child := ctx.contactID
	mother := ctx.relatedContactID
	father := createGraphContact(t, ctx, "Father")
	stepParent := createGraphContact(t, ctx, "StepParent")
	stepSibling := createGraphContact(t, ctx, "StepSibling")
	parentTypeID := seededGraphRelationshipTypeID(t, ctx, relKeyParent)
	spouseTypeID := seededGraphRelationshipTypeID(t, ctx, relKeySpouse)

	createGraphRelationship(t, ctx, mother, child, parentTypeID)
	createGraphRelationship(t, ctx, father, child, parentTypeID)
	createGraphRelationship(t, ctx, mother, father, spouseTypeID)
	createGraphRelationship(t, ctx, mother, stepParent, spouseTypeID)
	createGraphRelationship(t, ctx, stepParent, stepSibling, parentTypeID)

	graph, err := ctx.svc.GetContactGraph(child, ctx.vaultID, ctx.userID, "en")
	if err != nil {
		t.Fatalf("GetContactGraph: %v", err)
	}
	if graphRelationCount(graph, father, child, relKindParent, relKindChild, false) != 1 {
		t.Fatal("recorded father/child relation was not preserved exactly once")
	}
	if graphRelationCount(graph, mother, child, relKindParent, relKindChild, false) != 1 {
		t.Fatal("recorded mother/child relation was not preserved exactly once")
	}
	if graphRelationCount(graph, father, child, relKindStepParent, relKindStepChild, true) != 0 {
		t.Fatal("recorded parent was also misclassified as a step-parent")
	}
	if graphRelationCount(graph, stepParent, child, relKindStepParent, relKindStepChild, true) != 1 {
		t.Fatal("parent's non-parent spouse was not inferred as a step-parent")
	}
	if graphRelationCount(graph, mother, stepSibling, relKindStepParent, relKindStepChild, true) != 1 {
		t.Fatal("the reciprocal side of the blended family was not inferred")
	}
	if graphRelationCount(graph, child, stepSibling, relKindStepSibling, relKindStepSibling, true) != 1 {
		t.Fatal("children brought by different spouses were not inferred as step-siblings")
	}
}

func TestContactGraphInfersInLawsWithoutMisclassifyingThemAsStepFamily(t *testing.T) {
	ctx := setupRelationshipTestFull(t)
	person := ctx.contactID
	spouse := ctx.relatedContactID
	spouseParent := createGraphContact(t, ctx, "SpouseParent")
	parentTypeID := seededGraphRelationshipTypeID(t, ctx, relKeyParent)
	spouseTypeID := seededGraphRelationshipTypeID(t, ctx, relKeySpouse)

	createGraphRelationship(t, ctx, person, spouse, spouseTypeID)
	createGraphRelationship(t, ctx, spouseParent, spouse, parentTypeID)

	graph, err := ctx.svc.GetContactGraph(person, ctx.vaultID, ctx.userID, "en")
	if err != nil {
		t.Fatalf("GetContactGraph: %v", err)
	}
	if graphRelationCount(graph, spouseParent, person, relKindParentInLaw, relKindChildInLaw, true) != 1 {
		t.Fatal("spouse's parent was not inferred as a parent-in-law")
	}
	if graphRelationCount(graph, spouseParent, person, relKindStepParent, relKindStepChild, true) != 0 {
		t.Fatal("spouse's parent was misclassified as step-family")
	}

	for _, edge := range graph.Edges {
		if !((edge.Source == spouseParent && edge.Target == person) || (edge.Source == person && edge.Target == spouseParent)) {
			continue
		}
		for _, relation := range edge.Relations {
			parentKind, childKind := relation.SourceKind, relation.TargetKind
			parentLabel, childLabel := relation.SourceLabel, relation.TargetLabel
			if edge.Source != spouseParent {
				parentKind, childKind = childKind, parentKind
				parentLabel, childLabel = childLabel, parentLabel
			}
			if parentKind == relKindParentInLaw && childKind == relKindChildInLaw {
				if parentLabel != "parent-in-law" || childLabel != "child-in-law" {
					t.Fatalf("in-law labels = %q/%q, want parent-in-law/child-in-law", parentLabel, childLabel)
				}
				return
			}
		}
	}
	t.Fatal("in-law relation labels missing")
}

func TestContactGraphDoesNotInferStepParentFromNonSpouseRomance(t *testing.T) {
	ctx := setupRelationshipTestFull(t)
	child := ctx.contactID
	parent := ctx.relatedContactID
	partner := createGraphContact(t, ctx, "Partner")
	parentTypeID := seededGraphRelationshipTypeID(t, ctx, relKeyParent)
	significantOtherTypeID := seededGraphRelationshipTypeID(t, ctx, "seed.relationship_types.significant_other")

	createGraphRelationship(t, ctx, parent, child, parentTypeID)
	createGraphRelationship(t, ctx, parent, partner, significantOtherTypeID)

	graph, err := ctx.svc.GetContactGraph(child, ctx.vaultID, ctx.userID, "en")
	if err != nil {
		t.Fatalf("GetContactGraph: %v", err)
	}
	if graphRelationCount(graph, partner, child, relKindStepParent, relKindStepChild, true) != 0 {
		t.Fatal("a non-spouse romantic relationship was treated as a step-parent")
	}
}

func TestContactGraphInfersMultipleGenerationsAndSiblingClique(t *testing.T) {
	ctx := setupRelationshipTestFull(t)
	child := ctx.contactID
	siblingOne := ctx.relatedContactID
	siblingTwo := createGraphContact(t, ctx, "SiblingTwo")
	parent := createGraphContact(t, ctx, "Parent")
	secondParent := createGraphContact(t, ctx, "SecondParent")
	grandParent := createGraphContact(t, ctx, "GrandParent")
	greatGrandParent := createGraphContact(t, ctx, "GreatGrandParent")
	parentTypeID := seededGraphRelationshipTypeID(t, ctx, relKeyParent)

	createGraphRelationship(t, ctx, greatGrandParent, grandParent, parentTypeID)
	createGraphRelationship(t, ctx, grandParent, parent, parentTypeID)
	for _, descendant := range []string{child, siblingOne, siblingTwo} {
		createGraphRelationship(t, ctx, parent, descendant, parentTypeID)
		createGraphRelationship(t, ctx, secondParent, descendant, parentTypeID)
	}

	graph, err := ctx.svc.GetContactGraph(child, ctx.vaultID, ctx.userID, "en")
	if err != nil {
		t.Fatalf("GetContactGraph: %v", err)
	}
	if len(graph.Nodes) != 7 {
		t.Fatalf("nodes = %d, want the full seven-person lineage", len(graph.Nodes))
	}
	for _, descendant := range []string{child, siblingOne, siblingTwo} {
		if graphRelationCount(graph, grandParent, descendant, relKindGrandParent, relKindGrandChild, true) != 1 {
			t.Errorf("missing grandparent inference for descendant %s", descendant)
		}
		if graphRelationCount(graph, greatGrandParent, descendant, relKindAncestor, relKindDescendant, true) != 1 {
			t.Errorf("missing three-generation ancestor inference for descendant %s", descendant)
		}
		if generations := graphRelationGenerations(graph, greatGrandParent, descendant, relKindAncestor, relKindDescendant); generations != 3 {
			t.Errorf("ancestor generations = %d, want 3", generations)
		}
	}
	if graphRelationCount(graph, greatGrandParent, parent, relKindGrandParent, relKindGrandChild, true) != 1 {
		t.Fatal("missing grandparent inference in upper generation")
	}

	siblings := []string{child, siblingOne, siblingTwo}
	for i := 0; i < len(siblings); i++ {
		for j := i + 1; j < len(siblings); j++ {
			if graphRelationCount(graph, siblings[i], siblings[j], relKindSibling, relKindSibling, true) != 1 {
				t.Errorf("sibling pair %s/%s was not inferred exactly once", siblings[i], siblings[j])
			}
		}
	}
}

func TestContactGraphInfersUnclesAndMultipleCousins(t *testing.T) {
	ctx := setupRelationshipTestFull(t)
	child := ctx.contactID
	cousinOne := ctx.relatedContactID
	cousinTwo := createGraphContact(t, ctx, "CousinTwo")
	parent := createGraphContact(t, ctx, "Parent")
	aunt := createGraphContact(t, ctx, "Aunt")
	grandParent := createGraphContact(t, ctx, "GrandParent")
	parentTypeID := seededGraphRelationshipTypeID(t, ctx, relKeyParent)

	createGraphRelationship(t, ctx, grandParent, parent, parentTypeID)
	createGraphRelationship(t, ctx, grandParent, aunt, parentTypeID)
	createGraphRelationship(t, ctx, parent, child, parentTypeID)
	createGraphRelationship(t, ctx, aunt, cousinOne, parentTypeID)
	createGraphRelationship(t, ctx, aunt, cousinTwo, parentTypeID)

	graph, err := ctx.svc.GetContactGraph(child, ctx.vaultID, ctx.userID, "en")
	if err != nil {
		t.Fatalf("GetContactGraph: %v", err)
	}
	if graphRelationCount(graph, parent, aunt, relKindSibling, relKindSibling, true) != 1 {
		t.Fatal("shared-parent adults were not inferred as siblings")
	}
	if graphRelationCount(graph, aunt, child, relKindUncleAunt, relKindNephewNiece, true) != 1 {
		t.Fatal("parent's sibling was not inferred as aunt/uncle")
	}
	for _, cousin := range []string{cousinOne, cousinTwo} {
		if graphRelationCount(graph, child, cousin, relKindCousin, relKindCousin, true) != 1 {
			t.Errorf("cousin relation for %s was not inferred exactly once", cousin)
		}
		if graphRelationCount(graph, parent, cousin, relKindUncleAunt, relKindNephewNiece, true) != 1 {
			t.Errorf("reciprocal aunt/uncle branch for %s was not inferred", cousin)
		}
	}
	if graphRelationCount(graph, cousinOne, cousinTwo, relKindSibling, relKindSibling, true) != 1 {
		t.Fatal("multiple cousin nodes sharing a parent were not inferred as siblings")
	}
}

func TestContactGraphExplicitGrandparentOverridesEquivalentInference(t *testing.T) {
	ctx := setupRelationshipTestFull(t)
	child := ctx.contactID
	parent := ctx.relatedContactID
	grandParent := createGraphContact(t, ctx, "GrandParent")
	parentTypeID := seededGraphRelationshipTypeID(t, ctx, relKeyParent)
	grandParentTypeID := seededGraphRelationshipTypeID(t, ctx, relKeyGrandParent)

	createGraphRelationship(t, ctx, grandParent, parent, parentTypeID)
	createGraphRelationship(t, ctx, parent, child, parentTypeID)
	createGraphRelationship(t, ctx, grandParent, child, grandParentTypeID)

	graph, err := ctx.svc.GetContactGraph(child, ctx.vaultID, ctx.userID, "en")
	if err != nil {
		t.Fatalf("GetContactGraph: %v", err)
	}
	if graphRelationCount(graph, grandParent, child, relKindGrandParent, relKindGrandChild, false) != 1 {
		t.Fatal("explicit grandparent relation missing")
	}
	if graphRelationCount(graph, grandParent, child, relKindGrandParent, relKindGrandChild, true) != 0 {
		t.Fatal("equivalent inferred grandparent relation duplicated an explicit one")
	}
}

func TestContactGraphParentCycleTerminatesWithoutSelfInference(t *testing.T) {
	ctx := setupRelationshipTestFull(t)
	third := createGraphContact(t, ctx, "Third")
	parentTypeID := seededGraphRelationshipTypeID(t, ctx, relKeyParent)
	createGraphRelationship(t, ctx, ctx.contactID, ctx.relatedContactID, parentTypeID)
	createGraphRelationship(t, ctx, ctx.relatedContactID, third, parentTypeID)
	createGraphRelationship(t, ctx, third, ctx.contactID, parentTypeID)

	graph, err := ctx.svc.GetContactGraph(ctx.contactID, ctx.vaultID, ctx.userID, "en")
	if err != nil {
		t.Fatalf("GetContactGraph: %v", err)
	}
	for _, edge := range graph.Edges {
		if edge.Source == edge.Target {
			t.Fatalf("parent cycle produced a self edge: %#v", edge)
		}
	}
}
