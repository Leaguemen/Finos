package reimbursement

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

type CloudinaryStore struct {
	client *cloudinary.Cloudinary
	folder string
}

func NewCloudinaryStore(cloudName, apiKey, apiSecret, folder string) (*CloudinaryStore, error) {
	client, err := cloudinary.NewFromParams(cloudName, apiKey, apiSecret)
	if err != nil {
		return nil, fmt.Errorf("create Cloudinary client: %w", err)
	}
	client.Config.URL.Secure = true

	return &CloudinaryStore{
		client: client,
		folder: strings.Trim(strings.TrimSpace(folder), "/"),
	}, nil
}

func (store *CloudinaryStore) Upload(
	ctx context.Context,
	receipt ReceiptUpload,
) (ReceiptAsset, error) {
	publicID, err := randomPublicID()
	if err != nil {
		return ReceiptAsset{}, err
	}

	result, err := store.client.Upload.Upload(ctx, receipt.Reader, uploader.UploadParams{
		PublicID:       publicID,
		Folder:         store.folder,
		ResourceType:   api.Image.String(),
		Type:           api.Authenticated,
		AllowedFormats: api.CldAPIArray{"jpg", "jpeg", "png"},
		Headers:        "X-Robots-Tag: noindex",
	})
	if err != nil {
		return ReceiptAsset{}, fmt.Errorf("Cloudinary upload: %w", err)
	}
	if result == nil {
		return ReceiptAsset{}, errors.New("Cloudinary upload returned no result")
	}
	if result.Error.Message != "" {
		return ReceiptAsset{}, fmt.Errorf("Cloudinary upload rejected: %s", result.Error.Message)
	}
	if result.PublicID == "" || result.Format == "" {
		return ReceiptAsset{}, errors.New("Cloudinary upload returned incomplete asset metadata")
	}

	return ReceiptAsset{
		PublicID:     result.PublicID,
		AssetID:      result.AssetID,
		Format:       result.Format,
		ResourceType: result.ResourceType,
		DeliveryType: result.Type,
	}, nil
}

func (store *CloudinaryStore) Delete(ctx context.Context, asset ReceiptAsset) error {
	result, err := store.client.Upload.Destroy(ctx, uploader.DestroyParams{
		PublicID:     asset.PublicID,
		ResourceType: asset.ResourceType,
		Type:         asset.DeliveryType,
	})
	if err != nil {
		return fmt.Errorf("Cloudinary delete: %w", err)
	}
	if result != nil && result.Error.Message != "" {
		return fmt.Errorf("Cloudinary delete rejected: %s", result.Error.Message)
	}
	return nil
}

func (store *CloudinaryStore) DownloadURL(asset ReceiptAsset, expiresAt time.Time) (string, error) {
	url, err := store.client.Upload.PrivateDownloadURL(uploader.PrivateDownloadURLParams{
		PublicID:     asset.PublicID,
		Format:       asset.Format,
		DeliveryType: asset.DeliveryType,
		ExpiresAt:    &expiresAt,
		ResourceType: api.AssetType(asset.ResourceType),
	})
	if err != nil {
		return "", fmt.Errorf("Cloudinary private download URL: %w", err)
	}
	return url, nil
}

func randomPublicID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate receipt public ID: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}
