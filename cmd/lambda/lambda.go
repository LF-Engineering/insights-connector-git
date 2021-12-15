package main

import (
	"context"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	"log"
	"os"
)


func HandleRequest(ctx context.Context) {
	log.Println("lambda invoked")
	mysession := session.Must(session.NewSession())
	svc := ecs.New(mysession)
	runTaskInp := &ecs.RunTaskInput{

		Cluster:                  aws.String(os.Getenv("ECS_CLUSTER_NAME")),
		Count:                    aws.Int64(1),
		LaunchType:               aws.String("FARGATE"),
		NetworkConfiguration:    & ecs.NetworkConfiguration{AwsvpcConfiguration: &ecs.AwsVpcConfiguration{
			AssignPublicIp: aws.String("DISABLED"),
			SecurityGroups: nil,
			Subnets:        []*string{aws.String(os.Getenv("SUBNET_ID"))},
		}},
		Overrides:               & ecs.TaskOverride{
			ContainerOverrides:        []*ecs.ContainerOverride{{
				Name:                 aws.String("insights-git"),
				Command:              []*string{aws.String("./git"), aws.String("--git-url"), aws.String(os.Getenv("GIT_REPO")), aws.String("--git-es-url"), aws.String("ES_URL")},
			},

		}},
		TaskDefinition:           aws.String(os.Getenv("TASK_DEFINITION")),
	}
	log.Println("Input Prepared")
	output, err := svc.RunTask(runTaskInp)
	log.Println(output)
	log.Println(err)
}

func main() {
	lambda.Start(HandleRequest)
}