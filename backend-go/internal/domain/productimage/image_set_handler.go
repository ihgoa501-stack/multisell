package productimage

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lingmirror/backend-go/internal/imageservice"
	"github.com/lingmirror/backend-go/internal/response"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ImageSetHandler struct {
	db         *gorm.DB
	service    *ImageSetService
	image      ImageService
	governance *Service
}

func NewImageSetHandler(db *gorm.DB, image ImageService) *ImageSetHandler {
	return &ImageSetHandler{db: db, service: NewImageSetService(db), image: image, governance: NewService(db, zap.NewNop(), image)}
}

type createImageSetRequest struct {
	ListingID uint64                      `json:"listing_id"`
	Channel   string                      `json:"channel"`
	Locale    string                      `json:"locale"`
	Items     []createImageSetItemRequest `json:"items"`
}

type createImageSetItemRequest struct {
	TaskID  int64  `json:"task_id"`
	Role    string `json:"role"`
	Ordinal uint   `json:"ordinal"`
}

func (h *ImageSetHandler) Create(c *gin.Context) {
	owner, ok := ownerID(c)
	if !ok {
		return
	}
	var req createImageSetRequest
	if c.ShouldBindJSON(&req) != nil || req.ListingID == 0 || strings.TrimSpace(req.Channel) == "" || strings.TrimSpace(req.Locale) == "" || len(req.Items) == 0 {
		problem(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "listing_id, channel, locale and items are required")
		return
	}
	channel := strings.ToLower(strings.TrimSpace(req.Channel))
	if err := h.verifyListingAuthority(c.Request.Context(), owner, req.ListingID, channel); err != nil {
		h.writeAuthorityError(c, err)
		return
	}
	items := make([]ImageSetItemInput, len(req.Items))
	for i, input := range req.Items {
		if input.TaskID <= 0 {
			problem(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "each item requires a READY task_id")
			return
		}
		var task Task
		if err := h.db.WithContext(c.Request.Context()).Where("id = ? AND owner_id = ?", input.TaskID, owner).First(&task).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				problem(c, http.StatusNotFound, "NOT_FOUND", "image candidate not found")
				return
			}
			problem(c, http.StatusInternalServerError, "STORE_ERROR", "could not load image candidate")
			return
		}
		remote, err := h.verifyTaskLineage(c.Request.Context(), owner, &task)
		if err != nil {
			problem(c, http.StatusConflict, "CANDIDATE_NOT_READY", "only READY candidates with stored output bytes can enter an image set")
			return
		}
		purpose := imagePurposeForRole(input.Role)
		if err := verifyTaskChannelLineage(c.Request.Context(), h.db, owner, &task, channel, purpose); err != nil {
			problem(c, http.StatusConflict, "LINEAGE_INVALID", "task purpose, channel, or imported-asset restriction does not match this image-set item")
			return
		}
		if err := h.governance.VerifyPublicationGate(c.Request.Context(), owner, task.ID, remote.OutputBlobID, purpose, channel, req.Locale); err != nil {
			if errors.Is(err, ErrGateBlocked) {
				problem(c, http.StatusConflict, "GATE_BLOCKED", "valid exact-byte rights and five passed reviews are required")
				return
			}
			problem(c, http.StatusInternalServerError, "STORE_ERROR", "could not verify image publication gate")
			return
		}
		items[i] = ImageSetItemInput{Role: input.Role, Ordinal: input.Ordinal, Locale: strings.TrimSpace(req.Locale), Channel: channel, AssetSHA: remote.OutputBlobID, TaskID: task.ID, OutputBlobID: remote.OutputBlobID, TaskManifestHash: task.ManifestHash, Operation: task.Operation, Processor: task.Processor, ImageServiceJobID: task.ImageServiceJobID}
	}
	set, err := h.service.CreateDraft(c.Request.Context(), CreateImageSetInput{OwnerID: uint64(owner), ListingID: req.ListingID, Channel: req.Channel, Locale: req.Locale, Items: items})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, response.Result{Code: 0, Message: "ok", Data: set})
}

func (h *ImageSetHandler) Get(c *gin.Context) {
	owner, setID, ok := imageSetOwnerAndID(c)
	if !ok {
		return
	}
	set, err := h.service.Get(c.Request.Context(), uint64(owner), setID)
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, set)
}

func (h *ImageSetHandler) Freeze(c *gin.Context) {
	owner, setID, ok := imageSetOwnerAndID(c)
	if !ok {
		return
	}
	draft, err := h.service.Get(c.Request.Context(), uint64(owner), setID)
	if err != nil {
		h.writeError(c, err)
		return
	}
	if err := h.verifyListingAuthority(c.Request.Context(), owner, draft.ListingID, draft.Channel); err != nil {
		h.writeAuthorityError(c, err)
		return
	}
	for _, item := range draft.Items {
		var task Task
		if err := h.db.WithContext(c.Request.Context()).Where("id = ? AND owner_id = ?", item.TaskID, owner).First(&task).Error; err != nil {
			problem(c, http.StatusConflict, "LINEAGE_INVALID", "image task lineage is no longer verifiable")
			return
		}
		remote, verifyErr := h.verifyTaskLineage(c.Request.Context(), owner, &task)
		if verifyErr != nil || remote.OutputBlobID != item.OutputBlobID || task.ManifestHash != item.TaskManifestHash || task.Operation != item.Operation || task.Processor != item.Processor || task.ImageServiceJobID != item.ImageServiceJobID || verifyTaskChannelLineage(c.Request.Context(), h.db, owner, &task, draft.Channel, imagePurposeForRole(item.Role)) != nil {
			problem(c, http.StatusConflict, "LINEAGE_INVALID", "image task lineage changed before freeze")
			return
		}
		if err := h.governance.VerifyPublicationGate(c.Request.Context(), owner, task.ID, item.OutputBlobID, imagePurposeForRole(item.Role), draft.Channel, draft.Locale); err != nil {
			if errors.Is(err, ErrGateBlocked) {
				problem(c, http.StatusConflict, "GATE_BLOCKED", "rights or review changed before freeze")
				return
			}
			problem(c, http.StatusInternalServerError, "STORE_ERROR", "could not recheck image publication gate")
			return
		}
	}
	set, err := h.service.SelectAndFreeze(c.Request.Context(), uint64(owner), setID)
	if err != nil {
		h.writeError(c, err)
		return
	}
	response.Success(c, set)
}

func imagePurposeForRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "main":
		return "listing_main"
	case "ad_cover":
		return "advertising"
	default:
		return "listing_secondary"
	}
}

var errListingAuthority = errors.New("listing authority not proven")

func (h *ImageSetHandler) verifyListingAuthority(ctx context.Context, owner int64, listingID uint64, channel string) error {
	var count int64
	err := h.db.WithContext(ctx).Raw(`SELECT COUNT(*) FROM product_listing pl
JOIN platform p ON p.id = pl.platform_id
JOIN product_variant pv ON pv.sku_product_id = pl.product_id
JOIN product_master pm ON pm.id = pv.product_master_id
WHERE pl.id = ? AND pm.owner_id = ? AND LOWER(p.code) = ?`, listingID, owner, channel).Scan(&count).Error
	if err != nil {
		return err
	}
	if count == 0 {
		return errListingAuthority
	}
	return nil
}

func (h *ImageSetHandler) verifyTaskLineage(ctx context.Context, owner int64, task *Task) (*imageservice.Job, error) {
	if h.image == nil || task == nil || task.OwnerID != owner || task.ImageServiceJobID == "" {
		return nil, ErrImageSetInvalid
	}
	remote, err := h.image.GetJob(ctx, task.ImageServiceJobID)
	if err != nil {
		return nil, err
	}
	if task.Status != "READY" || remote.Status != "READY" || !verifyRemoteTaskIdentity(task, remote, owner) || isNonPublishableOutput(task, remote) || len(remote.OutputBlobID) != 64 || remote.OutputBlobID != task.OutputBlobID {
		return nil, ErrImageSetInvalid
	}
	return remote, nil
}

func (h *ImageSetHandler) writeAuthorityError(c *gin.Context, err error) {
	if errors.Is(err, errListingAuthority) {
		problem(c, http.StatusNotFound, "LISTING_NOT_FOUND", "listing not found for Owner and channel")
		return
	}
	problem(c, http.StatusInternalServerError, "STORE_ERROR", "could not verify listing authority")
}

func imageSetOwnerAndID(c *gin.Context) (int64, uint64, bool) {
	owner, ok := ownerID(c)
	if !ok {
		return 0, 0, false
	}
	id, err := strconv.ParseUint(c.Param("set_id"), 10, 64)
	if err != nil || id == 0 {
		problem(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid image set id")
		return 0, 0, false
	}
	return owner, id, true
}

func (h *ImageSetHandler) writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		problem(c, http.StatusNotFound, "NOT_FOUND", "image set not found")
	case errors.Is(err, ErrImageSetInvalid):
		problem(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
	case errors.Is(err, ErrImageSetFrozen), errors.Is(err, ErrImageSetNotFrozen):
		problem(c, http.StatusConflict, "IMAGE_SET_STATE_CONFLICT", err.Error())
	default:
		problem(c, http.StatusInternalServerError, "STORE_ERROR", "image set persistence failed")
	}
}
