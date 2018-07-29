package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/vektah/gqlgen/graphql"
	"github.com/vektah/gqlgen/neelance/errors"
	"github.com/vektah/gqlgen/neelance/query"
	"github.com/vektah/gqlgen/neelance/validation"
)

const (
	connectionInitMsg      = "connection_init"      // Client -> Server
	connectionTerminateMsg = "connection_terminate" // Client -> Server
	startMsg               = "start"                // Client -> Server
	stopMsg                = "stop"                 // Client -> Server
	connectionAckMsg       = "connection_ack"       // Server -> Client
	connectionErrorMsg     = "connection_error"     // Server -> Client
	dataMsg                = "data"                 // Server -> Client
	errorMsg               = "error"                // Server -> Client
	completeMsg            = "complete"             // Server -> Client
	//connectionKeepAliveMsg = "ka"                 // Server -> Client  TODO: keepalives
)

type operationMessage struct {
	Payload json.RawMessage `json:"payload,omitempty"`
	ID      string          `json:"id,omitempty"`
	Type    string          `json:"type"`
}

type wsConnection struct {
	ctx    context.Context
	conn   *websocket.Conn
	exec   graphql.ExecutableSchema
	active map[string]context.CancelFunc
	mu     sync.Mutex
	cfg    *Config
}

func connectWs(exec graphql.ExecutableSchema, w http.ResponseWriter, r *http.Request, cfg *Config) {
	ws, err := cfg.upgrader.Upgrade(w, r, http.Header{
		"Sec-Websocket-Protocol": []string{"graphql-ws"},
	})
	if err != nil {
		log.Printf("unable to upgrade %T to websocket %s: ", w, err.Error())
		sendErrorf(w, http.StatusBadRequest, "unable to upgrade")
		return
	}

	conn := wsConnection{
		active: map[string]context.CancelFunc{},
		exec:   exec,
		conn:   ws,
		ctx:    r.Context(),
		cfg:    cfg,
	}

	if !conn.init() {
		return
	}

	conn.run()
}

func (c *wsConnection) init() bool {
	message := c.readOp()
	if message == nil {
		c.close(websocket.CloseProtocolError, "decoding error")
		return false
	}

	switch message.Type {
	case connectionInitMsg:
		c.write(&operationMessage{Type: connectionAckMsg})
	case connectionTerminateMsg:
		c.close(websocket.CloseNormalClosure, "terminated")
		return false
	default:
		c.sendConnectionError("unexpected message %s", message.Type)
		c.close(websocket.CloseProtocolError, "unexpected message")
		return false
	}

	return true
}

func (c *wsConnection) write(msg *operationMessage) {
	c.mu.Lock()
	c.conn.WriteJSON(msg)
	c.mu.Unlock()
}

func (c *wsConnection) run() {
	for {
		message := c.readOp()
		if message == nil {
			return
		}

		switch message.Type {
		case startMsg:
			if !c.subscribe(message) {
				return
			}
		case stopMsg:
			c.mu.Lock()
			closer := c.active[message.ID]
			c.mu.Unlock()
			if closer == nil {
				c.sendError(message.ID, errors.Errorf("%s is not running, cannot stop", message.ID))
				continue
			}

			closer()
		case connectionTerminateMsg:
			c.close(websocket.CloseNormalClosure, "terminated")
			return
		default:
			c.sendConnectionError("unexpected message %s", message.Type)
			c.close(websocket.CloseProtocolError, "unexpected message")
			return
		}
	}
}

func (c *wsConnection) subscribe(message *operationMessage) bool {
	var reqParams params
	if err := json.Unmarshal(message.Payload, &reqParams); err != nil {
		c.sendConnectionError("invalid json")
		return false
	}

	doc, qErr := query.Parse(reqParams.Query)
	if qErr != nil {
		c.sendError(message.ID, qErr)
		return true
	}

	errs := validation.Validate(c.exec.Schema(), doc)
	if len(errs) != 0 {
		c.sendError(message.ID, errs...)
		return true
	}

	op, err := doc.GetOperation(reqParams.OperationName)
	if err != nil {
		c.sendError(message.ID, errors.Errorf("%s", err.Error()))
		return true
	}

	reqCtx := c.cfg.newRequestContext(doc, reqParams.Query, reqParams.Variables)
	ctx := graphql.WithRequestContext(c.ctx, reqCtx)

	if op.Type != query.Subscription {
		var result *graphql.Response
		if op.Type == query.Query {
			result = c.exec.Query(ctx, op)
		} else {
			result = c.exec.Mutation(ctx, op)
		}

		c.sendData(message.ID, result)
		c.write(&operationMessage{ID: message.ID, Type: completeMsg})
		return true
	}

	ctx, cancel := context.WithCancel(ctx)
	c.mu.Lock()
	c.active[message.ID] = cancel
	c.mu.Unlock()
	go func() {
		defer func() {
			if r := recover(); r != nil {
				userErr := reqCtx.Recover(ctx, r)
				c.sendError(message.ID, &errors.QueryError{Message: userErr.Error()})
			}
		}()
		next := c.exec.Subscription(ctx, op)
		for result := next(); result != nil; result = next() {
			c.sendData(message.ID, result)
		}

		c.write(&operationMessage{ID: message.ID, Type: completeMsg})

		c.mu.Lock()
		delete(c.active, message.ID)
		c.mu.Unlock()
		cancel()
	}()

	return true
}

func (c *wsConnection) sendData(id string, response *graphql.Response) {
	b, err := json.Marshal(response)
	if err != nil {
		c.sendError(id, errors.Errorf("unable to encode json response: %s", err.Error()))
		return
	}

	c.write(&operationMessage{Type: dataMsg, ID: id, Payload: b})
}

func (c *wsConnection) sendError(id string, errors ...*errors.QueryError) {
	var errs []error
	for _, err := range errors {
		errs = append(errs, err)
	}
	b, err := json.Marshal(errs)
	if err != nil {
		panic(err)
	}
	c.write(&operationMessage{Type: errorMsg, ID: id, Payload: b})
}

func (c *wsConnection) sendConnectionError(format string, args ...interface{}) {
	b, err := json.Marshal(&graphql.Error{Message: fmt.Sprintf(format, args...)})
	if err != nil {
		panic(err)
	}

	c.write(&operationMessage{Type: connectionErrorMsg, Payload: b})
}

func (c *wsConnection) readOp() *operationMessage {
	message := operationMessage{}
	if err := c.conn.ReadJSON(&message); err != nil {
		c.sendConnectionError("invalid json")
		return nil
	}
	return &message
}

func (c *wsConnection) close(closeCode int, message string) {
	c.mu.Lock()
	_ = c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(closeCode, message))
	c.mu.Unlock()
	_ = c.conn.Close()
}
