// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"

	"github.com/go-openapi/runtime"

	"github.com/future-architect/watertower/webapi/models"
)

// DeleteIndexDocIDOKCode is the HTTP code returned for type DeleteIndexDocIDOK
const DeleteIndexDocIDOKCode int = 200

/*DeleteIndexDocIDOK OK

swagger:response deleteIndexDocIdOK
*/
type DeleteIndexDocIDOK struct {

	/*
	  In: Body
	*/
	Payload *models.ModifyResponse `json:"body,omitempty"`
}

// NewDeleteIndexDocIDOK creates DeleteIndexDocIDOK with default headers values
func NewDeleteIndexDocIDOK() *DeleteIndexDocIDOK {

	return &DeleteIndexDocIDOK{}
}

// WithPayload adds the payload to the delete index doc Id o k response
func (o *DeleteIndexDocIDOK) WithPayload(payload *models.ModifyResponse) *DeleteIndexDocIDOK {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the delete index doc Id o k response
func (o *DeleteIndexDocIDOK) SetPayload(payload *models.ModifyResponse) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *DeleteIndexDocIDOK) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(200)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

// DeleteIndexDocIDBadRequestCode is the HTTP code returned for type DeleteIndexDocIDBadRequest
const DeleteIndexDocIDBadRequestCode int = 400

/*DeleteIndexDocIDBadRequest Bad Request

swagger:response deleteIndexDocIdBadRequest
*/
type DeleteIndexDocIDBadRequest struct {

	/*
	  In: Body
	*/
	Payload *DeleteIndexDocIDBadRequestBody `json:"body,omitempty"`
}

// NewDeleteIndexDocIDBadRequest creates DeleteIndexDocIDBadRequest with default headers values
func NewDeleteIndexDocIDBadRequest() *DeleteIndexDocIDBadRequest {

	return &DeleteIndexDocIDBadRequest{}
}

// WithPayload adds the payload to the delete index doc Id bad request response
func (o *DeleteIndexDocIDBadRequest) WithPayload(payload *DeleteIndexDocIDBadRequestBody) *DeleteIndexDocIDBadRequest {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the delete index doc Id bad request response
func (o *DeleteIndexDocIDBadRequest) SetPayload(payload *DeleteIndexDocIDBadRequestBody) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *DeleteIndexDocIDBadRequest) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(400)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

// DeleteIndexDocIDNotFoundCode is the HTTP code returned for type DeleteIndexDocIDNotFound
const DeleteIndexDocIDNotFoundCode int = 404

/*DeleteIndexDocIDNotFound Not Found

swagger:response deleteIndexDocIdNotFound
*/
type DeleteIndexDocIDNotFound struct {

	/*
	  In: Body
	*/
	Payload *DeleteIndexDocIDNotFoundBody `json:"body,omitempty"`
}

// NewDeleteIndexDocIDNotFound creates DeleteIndexDocIDNotFound with default headers values
func NewDeleteIndexDocIDNotFound() *DeleteIndexDocIDNotFound {

	return &DeleteIndexDocIDNotFound{}
}

// WithPayload adds the payload to the delete index doc Id not found response
func (o *DeleteIndexDocIDNotFound) WithPayload(payload *DeleteIndexDocIDNotFoundBody) *DeleteIndexDocIDNotFound {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the delete index doc Id not found response
func (o *DeleteIndexDocIDNotFound) SetPayload(payload *DeleteIndexDocIDNotFoundBody) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *DeleteIndexDocIDNotFound) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(404)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}
