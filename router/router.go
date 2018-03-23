// Copyright (c) TFG Co. All Rights Reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package router

import (
	"math/rand"
	"time"

	"github.com/topfreegames/pitaya/cluster"
	"github.com/topfreegames/pitaya/constants"
	"github.com/topfreegames/pitaya/logger"
	"github.com/topfreegames/pitaya/protos"
	"github.com/topfreegames/pitaya/route"
	"github.com/topfreegames/pitaya/session"
)

var log = logger.Log

// Router struct
type Router struct {
	serviceDiscovery cluster.ServiceDiscovery
	routesMap        map[string]func(
		session *session.Session,
		route *route.Route,
		servers []*cluster.Server,
	) (*cluster.Server, error)
}

// New returns the router
func New() *Router {
	return &Router{
		routesMap: make(map[string]func(
			session *session.Session,
			route *route.Route,
			servers []*cluster.Server,
		) (*cluster.Server, error)),
	}
}

// SetServiceDiscovery sets the sd client
func (r *Router) SetServiceDiscovery(sd cluster.ServiceDiscovery) {
	r.serviceDiscovery = sd
}

func (r *Router) defaultRoute(
	svType string,
	servers []*cluster.Server,
) (*cluster.Server, error) {
	s := rand.NewSource(time.Now().Unix())
	rnd := rand.New(s)
	server := servers[rnd.Intn(len(servers))]
	return server, nil
}

// Route gets the right server to use in the call
func (r *Router) Route(
	rpcType protos.RPCType,
	svType string,
	session *session.Session,
	route *route.Route,
) (*cluster.Server, error) {
	if r.serviceDiscovery == nil {
		return nil, constants.ErrServiceDiscoveryNotInitialized
	}
	serversOfType, err := r.serviceDiscovery.GetServersByType(svType)
	if err != nil {
		return nil, err
	}
	if rpcType == protos.RPCType_User {
		return r.defaultRoute(svType, serversOfType)
	}
	routeFunc, ok := r.routesMap[svType]
	if !ok {
		log.Debugf("no specific route for svType: %s, using default route", svType)
		return r.defaultRoute(svType, serversOfType)
	}
	return routeFunc(session, route, serversOfType)
}

// AddRoute adds a routing function to a server type
// TODO calling this method with the server already running is VERY dangerous
func (r *Router) AddRoute(
	serverType string,
	routingFunction func(
		session *session.Session,
		route *route.Route,
		servers []*cluster.Server,
	) (*cluster.Server, error),
) {
	if _, ok := r.routesMap[serverType]; ok {
		log.Warnf("overriding the route to svType %s", serverType)
	}
	r.routesMap[serverType] = routingFunction
}
