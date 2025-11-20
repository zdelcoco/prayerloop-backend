package controllers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/PrayerLoop/initializers"
	"github.com/PrayerLoop/models"
	"github.com/doug-martin/goqu/v9"
	"github.com/gin-gonic/gin"
)

// GetUserCategories - Get all categories for a user
func GetUserCategories(c *gin.Context) {
	currentUser := c.MustGet("currentUser").(models.UserProfile)
	userID, err := strconv.Atoi(c.Param("user_profile_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Users can only get their own categories
	if currentUser.User_Profile_ID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only view your own categories"})
		return
	}

	var categories []models.PrayerCategory
	err = initializers.DB.From("prayer_category").
		Where(goqu.C("category_type").Eq("user"), goqu.C("category_type_id").Eq(userID)).
		Order(goqu.C("display_sequence").Asc()).
		ScanStructs(&categories)

	if err != nil {
		log.Println("Error fetching user categories:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch categories", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, categories)
}

// GetGroupCategories - Get all categories for a group
func GetGroupCategories(c *gin.Context) {
	currentUser := c.MustGet("currentUser").(models.UserProfile)
	groupID, err := strconv.Atoi(c.Param("group_profile_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
		return
	}

	// Verify user is a member of the group
	var membership models.UserGroup
	found, err := initializers.DB.From("user_group").
		Where(goqu.C("user_profile_id").Eq(currentUser.User_Profile_ID), goqu.C("group_profile_id").Eq(groupID)).
		ScanStruct(&membership)

	if err != nil || !found {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not a member of this group"})
		return
	}

	var categories []models.PrayerCategory
	err = initializers.DB.From("prayer_category").
		Where(goqu.C("category_type").Eq("group"), goqu.C("category_type_id").Eq(groupID)).
		Order(goqu.C("display_sequence").Asc()).
		ScanStructs(&categories)

	if err != nil {
		log.Println("Error fetching group categories:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch categories", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, categories)
}

// CreateUserCategory - Create a new category for a user
func CreateUserCategory(c *gin.Context) {
	currentUser := c.MustGet("currentUser").(models.UserProfile)
	userID, err := strconv.Atoi(c.Param("user_profile_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Users can only create categories for themselves
	if currentUser.User_Profile_ID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only create categories for yourself"})
		return
	}

	var body models.PrayerCategoryCreate
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get max sequence for this user
	var maxSeq int
	_, err = initializers.DB.From("prayer_category").
		Select(goqu.COALESCE(goqu.MAX("display_sequence"), -1)).
		Where(goqu.C("category_type").Eq("user"), goqu.C("category_type_id").Eq(userID)).
		ScanVal(&maxSeq)

	if err != nil {
		log.Println("Error getting max sequence:", err)
		maxSeq = -1
	}

	category := models.PrayerCategory{
		Category_Type:    "user",
		Category_Type_ID: userID,
		Category_Name:    body.Category_Name,
		Category_Color:   body.Category_Color,
		Display_Sequence: maxSeq + 1,
		Created_By:       currentUser.User_Profile_ID,
		Updated_By:       currentUser.User_Profile_ID,
	}

	result, err := initializers.DB.Insert("prayer_category").
		Rows(category).
		Executor().
		Exec()

	if err != nil {
		log.Println("Error creating category:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create category", "details": err.Error()})
		return
	}

	categoryID, _ := result.LastInsertId()
	category.Prayer_Category_ID = int(categoryID)

	c.JSON(http.StatusCreated, category)
}

// CreateGroupCategory - Create a new category for a group
func CreateGroupCategory(c *gin.Context) {
	currentUser := c.MustGet("currentUser").(models.UserProfile)
	groupID, err := strconv.Atoi(c.Param("group_profile_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
		return
	}

	// Verify user is a member of the group
	var membership models.UserGroup
	found, err := initializers.DB.From("user_group").
		Where(goqu.C("user_profile_id").Eq(currentUser.User_Profile_ID), goqu.C("group_profile_id").Eq(groupID)).
		ScanStruct(&membership)

	if err != nil || !found {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not a member of this group"})
		return
	}

	var body models.PrayerCategoryCreate
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get max sequence for this group
	var maxSeq int
	_, err = initializers.DB.From("prayer_category").
		Select(goqu.COALESCE(goqu.MAX("display_sequence"), -1)).
		Where(goqu.C("category_type").Eq("group"), goqu.C("category_type_id").Eq(groupID)).
		ScanVal(&maxSeq)

	if err != nil {
		log.Println("Error getting max sequence:", err)
		maxSeq = -1
	}

	category := models.PrayerCategory{
		Category_Type:    "group",
		Category_Type_ID: groupID,
		Category_Name:    body.Category_Name,
		Category_Color:   body.Category_Color,
		Display_Sequence: maxSeq + 1,
		Created_By:       currentUser.User_Profile_ID,
		Updated_By:       currentUser.User_Profile_ID,
	}

	result, err := initializers.DB.Insert("prayer_category").
		Rows(category).
		Executor().
		Exec()

	if err != nil {
		log.Println("Error creating category:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create category", "details": err.Error()})
		return
	}

	categoryID, _ := result.LastInsertId()
	category.Prayer_Category_ID = int(categoryID)

	c.JSON(http.StatusCreated, category)
}

// UpdateCategory - Update a category (name and/or color)
func UpdateCategory(c *gin.Context) {
	currentUser := c.MustGet("currentUser").(models.UserProfile)
	categoryID, err := strconv.Atoi(c.Param("prayer_category_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
		return
	}

	// Get the category to verify ownership
	var category models.PrayerCategory
	found, err := initializers.DB.From("prayer_category").
		Where(goqu.C("prayer_category_id").Eq(categoryID)).
		ScanStruct(&category)

	if err != nil || !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
		return
	}

	// Verify user can update this category
	if category.Category_Type == "user" && category.Category_Type_ID != currentUser.User_Profile_ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only update your own categories"})
		return
	}

	if category.Category_Type == "group" {
		// Verify user is a member of the group
		var membership models.UserGroup
		found, err := initializers.DB.From("user_group").
			Where(goqu.C("user_profile_id").Eq(currentUser.User_Profile_ID), goqu.C("group_profile_id").Eq(category.Category_Type_ID)).
			ScanStruct(&membership)

		if err != nil || !found {
			c.JSON(http.StatusForbidden, gin.H{"error": "You are not a member of this group"})
			return
		}
	}

	var body models.PrayerCategoryUpdate
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	update := initializers.DB.Update("prayer_category").
		Set(goqu.Record{
			"category_name":  body.Category_Name,
			"category_color": body.Category_Color,
			"updated_by":     currentUser.User_Profile_ID,
		}).
		Where(goqu.C("prayer_category_id").Eq(categoryID))

	_, err = update.Executor().Exec()
	if err != nil {
		log.Println("Error updating category:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update category", "details": err.Error()})
		return
	}

	// Fetch updated category
	found, err = initializers.DB.From("prayer_category").
		Where(goqu.C("prayer_category_id").Eq(categoryID)).
		ScanStruct(&category)

	if err != nil || !found {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch updated category"})
		return
	}

	c.JSON(http.StatusOK, category)
}

// DeleteCategory - Delete a category
func DeleteCategory(c *gin.Context) {
	currentUser := c.MustGet("currentUser").(models.UserProfile)
	categoryID, err := strconv.Atoi(c.Param("prayer_category_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
		return
	}

	// Get the category to verify ownership
	var category models.PrayerCategory
	found, err := initializers.DB.From("prayer_category").
		Where(goqu.C("prayer_category_id").Eq(categoryID)).
		ScanStruct(&category)

	if err != nil || !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
		return
	}

	// Verify user can delete this category
	if category.Category_Type == "user" && category.Category_Type_ID != currentUser.User_Profile_ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only delete your own categories"})
		return
	}

	if category.Category_Type == "group" {
		// Verify user is a member of the group
		var membership models.UserGroup
		found, err := initializers.DB.From("user_group").
			Where(goqu.C("user_profile_id").Eq(currentUser.User_Profile_ID), goqu.C("group_profile_id").Eq(category.Category_Type_ID)).
			ScanStruct(&membership)

		if err != nil || !found {
			c.JSON(http.StatusForbidden, gin.H{"error": "You are not a member of this group"})
			return
		}
	}

	// Delete the category (cascade will remove prayer_category_item entries)
	_, err = initializers.DB.Delete("prayer_category").
		Where(goqu.C("prayer_category_id").Eq(categoryID)).
		Executor().
		Exec()

	if err != nil {
		log.Println("Error deleting category:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete category", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Category deleted successfully"})
}

// ReorderUserCategories - Reorder user's categories
func ReorderUserCategories(c *gin.Context) {
	currentUser := c.MustGet("currentUser").(models.UserProfile)
	userID, err := strconv.Atoi(c.Param("user_profile_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Users can only reorder their own categories
	if currentUser.User_Profile_ID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only reorder your own categories"})
		return
	}

	var body models.PrayerCategoryReorder
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update display_sequence for each category
	for i, categoryID := range body.Category_IDs {
		_, err = initializers.DB.Update("prayer_category").
			Set(goqu.Record{"display_sequence": i}).
			Where(
				goqu.C("prayer_category_id").Eq(categoryID),
				goqu.C("category_type").Eq("user"),
				goqu.C("category_type_id").Eq(userID),
			).
			Executor().
			Exec()

		if err != nil {
			log.Println("Error reordering category:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reorder categories", "details": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Categories reordered successfully"})
}

// ReorderGroupCategories - Reorder group's categories
func ReorderGroupCategories(c *gin.Context) {
	currentUser := c.MustGet("currentUser").(models.UserProfile)
	groupID, err := strconv.Atoi(c.Param("group_profile_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid group ID"})
		return
	}

	// Verify user is a member of the group
	var membership models.UserGroup
	found, err := initializers.DB.From("user_group").
		Where(goqu.C("user_profile_id").Eq(currentUser.User_Profile_ID), goqu.C("group_profile_id").Eq(groupID)).
		ScanStruct(&membership)

	if err != nil || !found {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not a member of this group"})
		return
	}

	var body models.PrayerCategoryReorder
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update display_sequence for each category
	for i, categoryID := range body.Category_IDs {
		_, err = initializers.DB.Update("prayer_category").
			Set(goqu.Record{"display_sequence": i}).
			Where(
				goqu.C("prayer_category_id").Eq(categoryID),
				goqu.C("category_type").Eq("group"),
				goqu.C("category_type_id").Eq(groupID),
			).
			Executor().
			Exec()

		if err != nil {
			log.Println("Error reordering category:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reorder categories", "details": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Categories reordered successfully"})
}

// AddPrayerToCategory - Add a prayer to a category
func AddPrayerToCategory(c *gin.Context) {
	currentUser := c.MustGet("currentUser").(models.UserProfile)
	categoryID, err := strconv.Atoi(c.Param("prayer_category_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category ID"})
		return
	}

	prayerAccessID, err := strconv.Atoi(c.Param("prayer_access_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid prayer access ID"})
		return
	}

	// Get category to verify type and ownership
	var category models.PrayerCategory
	found, err := initializers.DB.From("prayer_category").
		Where(goqu.C("prayer_category_id").Eq(categoryID)).
		ScanStruct(&category)

	if err != nil || !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
		return
	}

	// Get prayer access to verify type and ownership
	var prayerAccess models.PrayerAccess
	found, err = initializers.DB.From("prayer_access").
		Where(goqu.C("prayer_access_id").Eq(prayerAccessID)).
		ScanStruct(&prayerAccess)

	if err != nil || !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Prayer not found"})
		return
	}

	// Validate type matching
	if category.Category_Type == "user" {
		if prayerAccess.Access_Type != "user" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "User categories can only contain personal prayers"})
			return
		}
		if category.Category_Type_ID != currentUser.User_Profile_ID || prayerAccess.Access_Type_ID != currentUser.User_Profile_ID {
			c.JSON(http.StatusForbidden, gin.H{"error": "You can only categorize your own prayers"})
			return
		}
	}

	if category.Category_Type == "group" {
		if prayerAccess.Access_Type != "group" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Group categories can only contain group prayers"})
			return
		}
		if category.Category_Type_ID != prayerAccess.Access_Type_ID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Prayer does not belong to this group"})
			return
		}

		// Verify user is a member
		var membership models.UserGroup
		found, err := initializers.DB.From("user_group").
			Where(goqu.C("user_profile_id").Eq(currentUser.User_Profile_ID), goqu.C("group_profile_id").Eq(category.Category_Type_ID)).
			ScanStruct(&membership)

		if err != nil || !found {
			c.JSON(http.StatusForbidden, gin.H{"error": "You are not a member of this group"})
			return
		}
	}

	// Check if prayer already has a category
	var existing models.PrayerCategoryItem
	existingFound, err := initializers.DB.From("prayer_category_item").
		Where(goqu.C("prayer_access_id").Eq(prayerAccessID)).
		ScanStruct(&existing)

	if err == nil && existingFound {
		// Already has a category - update it
		_, err = initializers.DB.Update("prayer_category_item").
			Set(goqu.Record{"prayer_category_id": categoryID}).
			Where(goqu.C("prayer_access_id").Eq(prayerAccessID)).
			Executor().
			Exec()

		if err != nil {
			log.Println("Error updating prayer category:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update prayer category", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Prayer moved to category"})
		return
	}

	// Create new category item
	item := models.PrayerCategoryItem{
		Prayer_Category_ID: categoryID,
		Prayer_Access_ID:   prayerAccessID,
		Created_By:         currentUser.User_Profile_ID,
	}

	_, err = initializers.DB.Insert("prayer_category_item").
		Rows(item).
		Executor().
		Exec()

	if err != nil {
		log.Println("Error adding prayer to category:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add prayer to category", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Prayer added to category"})
}

// RemovePrayerFromCategory - Remove a prayer from its category
func RemovePrayerFromCategory(c *gin.Context) {
	currentUser := c.MustGet("currentUser").(models.UserProfile)
	prayerAccessID, err := strconv.Atoi(c.Param("prayer_access_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid prayer access ID"})
		return
	}

	// Get the prayer category item
	var item models.PrayerCategoryItem
	found, err := initializers.DB.From("prayer_category_item").
		Where(goqu.C("prayer_access_id").Eq(prayerAccessID)).
		ScanStruct(&item)

	if err != nil || !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Prayer is not in any category"})
		return
	}

	// Get category to verify ownership
	var category models.PrayerCategory
	found, err = initializers.DB.From("prayer_category").
		Where(goqu.C("prayer_category_id").Eq(item.Prayer_Category_ID)).
		ScanStruct(&category)

	if err != nil || !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
		return
	}

	// Verify permission
	if category.Category_Type == "user" && category.Category_Type_ID != currentUser.User_Profile_ID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only remove prayers from your own categories"})
		return
	}

	if category.Category_Type == "group" {
		var membership models.UserGroup
		found, err := initializers.DB.From("user_group").
			Where(goqu.C("user_profile_id").Eq(currentUser.User_Profile_ID), goqu.C("group_profile_id").Eq(category.Category_Type_ID)).
			ScanStruct(&membership)

		if err != nil || !found {
			c.JSON(http.StatusForbidden, gin.H{"error": "You are not a member of this group"})
			return
		}
	}

	// Remove the item
	_, err = initializers.DB.Delete("prayer_category_item").
		Where(goqu.C("prayer_access_id").Eq(prayerAccessID)).
		Executor().
		Exec()

	if err != nil {
		log.Println("Error removing prayer from category:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove prayer from category", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Prayer removed from category"})
}
