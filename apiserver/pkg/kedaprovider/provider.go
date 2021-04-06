package externalscaler

// import (
// 	"context"
// 	"fmt"
// 	"log"
// 	"net"
// 	"time"

// 	"github.com/devjoes/github-runner-autoscaler/apiserver/pkg/host"
// 	"google.golang.org/grpc"
// 	"google.golang.org/grpc/reflection"
// 	"k8s.io/klog/v2"
// )

// //TODO: remove - unused
// type ExternalScaler struct{ orchestrator *host.Host }

// func (e *ExternalScaler) IsActive(ctx context.Context, scaledObject *ScaledObjectRef) (*IsActiveResponse, error) {
// 	name := scaledObject.Name
// 	fmt.Printf("IsActive: %s %v\n", name, *scaledObject)
// 	queueLength, _, err := e.orchestrator.QueryMetric(name)
// 	found := true
// 	if err != nil {
// 		if err.Error() == host.MetricErrNotFound {
// 			found = false
// 		}
// 		fmt.Printf("Error whilst processing metric %s\n%s\n", name, err.Error())
// 		return nil, err
// 	}
// 	fmt.Printf("IsActive: %s %t\n", name, found)
// 	return &IsActiveResponse{
// 		Result: found && queueLength > 0,
// 	}, nil
// }

// func (e *ExternalScaler) StreamIsActive(scaledObject *ScaledObjectRef, epsServer ExternalScaler_StreamIsActiveServer) error {
// 	name := scaledObject.Name
// 	fmt.Printf("StreamIsActive: %s %v\n", name, *scaledObject)
// 	for {
// 		select {
// 		case <-epsServer.Context().Done():
// 			// call cancelled
// 			return nil
// 		case <-time.Tick(time.Hour * 1):
// 			queueLength, _, err := e.orchestrator.QueryMetric(name)
// 			if err != nil {
// 				err = fmt.Errorf("Errored whilst getting metric %s. %s", name, err.Error())
// 			} else {
// 				err = epsServer.Send(&IsActiveResponse{
// 					Result: queueLength > 0,
// 				})
// 			}
// 			if err != nil {
// 				klog.Errorln(err)
// 			}
// 		}
// 	}
// }

// func (e *ExternalScaler) GetMetricSpec(ctx context.Context, scaledObject *ScaledObjectRef) (*GetMetricSpecResponse, error) {
// 	name := scaledObject.Name
// 	fmt.Printf("GetMetricSpec: %s %v\n", name, *scaledObject)
// 	queue, _, err := e.orchestrator.QueryMetric(name)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &GetMetricSpecResponse{
// 		MetricSpecs: []*MetricSpec{{
// 			MetricName: name,
// 			TargetSize: int64(queue),
// 		}},
// 	}, nil
// }

// func (e *ExternalScaler) GetMetrics(_ context.Context, metricRequest *GetMetricsRequest) (*GetMetricsResponse, error) {
// 	name := metricRequest.MetricName
// 	fmt.Printf("GetMetrics: %s %v\n", name, *metricRequest)
// 	queue, _, err := e.orchestrator.QueryMetric(name)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &GetMetricsResponse{
// 		MetricValues: []*MetricValue{{
// 			MetricName:  name,
// 			MetricValue: int64(queue),
// 		}},
// 	}, nil
// }

// func NewKedaProvider(orchestrator *host.Host) {
// 	grpcServer := grpc.NewServer()
// 	reflection.Register(grpcServer) //TODO: remove
// 	lis, _ := net.Listen("tcp", ":6000")
// 	server := ExternalScaler{orchestrator: orchestrator}
// 	RegisterExternalScalerServer(grpcServer, &server)

// 	fmt.Println("listenting on :6000")
// 	if err := grpcServer.Serve(lis); err != nil {
// 		log.Fatal(err)
// 	}
// }
