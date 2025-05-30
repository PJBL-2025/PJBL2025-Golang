package services

import (
	"fmt"
	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/snap"
	"restfull-api-pjbl-2025/app/repositories"
	"restfull-api-pjbl-2025/helper"
	"restfull-api-pjbl-2025/model"
	"restfull-api-pjbl-2025/model/dto"
	"strconv"
)

type CheckoutServiceImpl struct {
	checkoutRepository repositories.CheckoutsRepository
	snapClient         *snap.Client
}

func NewCheckoutServiceImpl(checkoutRepository repositories.CheckoutsRepository, snapClient *snap.Client) *CheckoutServiceImpl {
	return &CheckoutServiceImpl{
		checkoutRepository: checkoutRepository,
		snapClient:         snapClient,
	}
}

func (service *CheckoutServiceImpl) CreateOrderUser(userId int, checkout *map[string]interface{}) (string, string, error) {
	fmt.Println("CreateOrderUser")
	orderId := helper.GenerateOrderID()
	totalPrice := int((*checkout)["total_price"].(float64))
	addressId := int((*checkout)["address_id"].(float64))
	productCheckout := (*checkout)["product_checkout"].([]interface{})

	checkoutId, err := service.checkoutRepository.CreateCheckout(userId, totalPrice, addressId, orderId)
	if err != nil {
		return "", "", err
	}

	fmt.Println(checkoutId)
	err = service.CreateOrderCustom(productCheckout, checkoutId)
	if err != nil {
		return "", "", err
	}

	req := &snap.Request{
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:  strconv.Itoa(orderId),
			GrossAmt: int64(totalPrice),
		},
	}
	snapResponse, err := service.snapClient.CreateTransaction(req)
	fmt.Println(err)
	err = service.checkoutRepository.InsertSnapToken(checkoutId, snapResponse.Token)
	if err != nil {
		return "", "", err
	}
	return snapResponse.RedirectURL, snapResponse.Token, nil
}

func (service *CheckoutServiceImpl) CreateOrderCustom(productCheckout []interface{}, checkoutId int) error {
	for i := range productCheckout {
		product := productCheckout[i].(map[string]interface{})

		if product["type"] == "reguler" {
			productId := int(product["product_id"].(float64))
			orderProduct := model.ProductCheckout{
				Quantity:   int(product["quantity"].(float64)),
				Size:       product["size"].(string),
				Color:      product["color"].(string),
				Type:       product["type"].(string),
				Price:      int(product["price"].(float64)),
				ProductId:  &productId,
				CheckoutId: checkoutId,
			}

			productCustomId, err := service.checkoutRepository.CreateOrder(&orderProduct)
			if err != nil {
				return err
			}
			fmt.Println(productCustomId)
		}

		if product["type"] == "custom" {
			productId := int(product["product_id"].(float64))
			orderProduct := model.ProductCheckout{
				Quantity:   int(product["quantity"].(float64)),
				Size:       product["size"].(string),
				Color:      product["color"].(string),
				Type:       product["type"].(string),
				Price:      int(product["price"].(float64)),
				ProductId:  &productId,
				CheckoutId: checkoutId,
			}

			productCustomId, err := service.checkoutRepository.CreateOrder(&orderProduct)
			err = service.checkoutRepository.CreateProductCustom(product["product_custom"].(map[string]interface{}), productCustomId)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (service *CheckoutServiceImpl) UpdateStatusCheckout(checkout *dto.RequestUpdateCheckout) error {
	fmt.Println("cek")
	if checkout.Status == "processing" {
		deliveryId, err := service.checkoutRepository.CreateDelivery(checkout.CheckoutId)
		if err != nil {
			return err
		}

		fmt.Println("test")
		err = service.checkoutRepository.CreateStatusDelivery("Pesanan Sedang Di Kemas", deliveryId)
		if err != nil {
			return err
		}
	}

	err := service.checkoutRepository.UpdateStatusCheckout(checkout)
	if err != nil {
		return err
	}

	return nil
}

func (service *CheckoutServiceImpl) SetDelivery(delivery *dto.SetDelivery, deliveryId int) error {
	return service.checkoutRepository.SetDelivery(delivery, deliveryId)
}

func (service *CheckoutServiceImpl) SetStatusDelivery(status map[string]interface{}) error {
	return service.checkoutRepository.SetStatusDelivery(status)
}
func (service *CheckoutServiceImpl) GetCheckout(param string, userId int) ([]map[string]interface{}, error) {
	var flatData []map[string]interface{}
	var err error

	if param == "pending" {
		flatData, err = service.checkoutRepository.GetCheckoutPending(param, userId)
		if err != nil {
			return nil, err
		}

		grouped := map[int]map[string]interface{}{}

		for _, row := range flatData {
			id := int(row["id"].(int32))

			if _, exists := grouped[id]; !exists {
				grouped[id] = map[string]interface{}{
					"id":          id,
					"order_id":    row["order_id"],
					"total_price": row["total_price"],
					"product":     []map[string]interface{}{},
				}
			}
			product := map[string]interface{}{
				"name":       row["name"],
				"price":      row["price"],
				"quantity":   row["quantity"],
				"type":       row["type"],
				"image_path": row["image_path"],
			}
			grouped[id]["product"] = append(grouped[id]["product"].([]map[string]interface{}), product)
		}

		var result []map[string]interface{}
		for _, item := range grouped {
			result = append(result, item)
		}

		return result, nil

	} else {
		return service.checkoutRepository.GetCheckoutNotPending(param, userId)
	}
}

func (service *CheckoutServiceImpl) GetDetailCheckoutProduct(productCheckoutId int) (map[string]interface{}, error) {
	data, err := service.checkoutRepository.GetDetailProductCheckout(productCheckoutId)
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("data not found")
	}

	result := map[string]interface{}{
		"name_product":    data[0]["product_name"],
		"price":           data[0]["price"],
		"date":            data[0]["date"],
		"name_user":       data[0]["user_name"],
		"checkout_status": data[0]["checkout_status"],
		"image_path":      data[0]["image_path"],
		"order_id":        data[0]["order_id"],
		"product":         []map[string]interface{}{},
	}

	var products []map[string]interface{}
	for _, item := range data {
		productItem := map[string]interface{}{
			"status": item["delivery_status"],
		}
		products = append(products, productItem)
	}

	result["product"] = products
	return result, nil
}
