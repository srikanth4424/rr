func deleteListeners(svc *elbv2.ELBV2, lbName string) error {
    // Get load balancer ARN
    lbDesc, err := svc.DescribeLoadBalancers(&elbv2.DescribeLoadBalancersInput{
        Names: []*string{aws.String(lbName)},
    })
    if err != nil {
        return err
    }

    // Delete all listeners
    listeners, err := svc.DescribeListeners(&elbv2.DescribeListenersInput{
        LoadBalancerArn: lbDesc.LoadBalancers[0].LoadBalancerArn,
    })
    if err != nil {
        return err
    }

    for _, listener := range listeners.Listeners {
        _, err := svc.DeleteListener(&elbv2.DeleteListenerInput{
            ListenerArn: listener.ListenerArn,
        })
        if err != nil {
            return err
        }
    }
    return nil
}
