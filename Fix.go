### **1. Imports Section Update**
**Add**:  
```go
"github.com/aws/aws-sdk-go/service/elbv2"
```

---

### **2. AWS Client Initialization**  
**File**: Inside `ginkgo.It("manually deleted "+cioServiceName+" load balancer should be recreated", ...)`  
**Line**: After creating `awsSession`  
**Add**:  
```go
// Initialize all AWS clients
elbSvc := elb.New(awsSession)
elbv2Svc := elbv2.New(awsSession) // For Target Groups
ec2Svc := ec2.New(awsSession)     // For Security Groups
```

---

### **3. Target Group Cleanup Function**  
**File**: At the bottom of the file (after `deleteSecGroupReferencesToOrphans`)  
**Add**:  
```go
// cleanupTargetGroups deletes target groups associated with a Classic Load Balancer
func cleanupTargetGroups(elbv2Svc *elbv2.ELBV2, lbName string) error {
    output, err := elbv2Svc.DescribeTargetGroups(&elbv2.DescribeTargetGroupsInput{
        LoadBalancerArn: aws.String(lbName),
    })
    if err != nil {
        return fmt.Errorf("failed to list target groups: %v", err)
    }

    // Delete all associated target groups
    for _, tg := range output.TargetGroups {
        _, err := elbv2Svc.DeleteTargetGroup(&elbv2.DeleteTargetGroupInput{
            TargetGroupArn: tg.TargetGroupArn,
        })
        if err != nil {
            return fmt.Errorf("failed to delete target group %s: %v", *tg.TargetGroupArn, err)
        }
        log.Printf("Deleted target group: %s", *tg.TargetGroupArn)
    }
    return nil
}
```

---

### **4. Enhanced Cleanup Logic**  
**File**: Inside the AWS test block (`if provider == "aws"`)  
**Line**: After deleting the security groups  
**Add**:  
```go
// Delete target groups
ginkgo.By("Cleaning up orphaned target groups")
err = cleanupTargetGroups(elbv2Svc, oldLBName)
if err != nil {
    log.Printf("Warning: Target group cleanup failed: %v", err)
}
```

---

### **5. Critical Fixes in Existing Code**  
**File**: AWS LB deletion logic  
**Line**: Where you initialize `lb`  
**Change**:  
```diff
- lb := elb.New(awsSession)
+ lb := elbSvc  // Use the pre-initialized client
```

---

### **6. Eventual Consistency Handling**  
**File**: After deleting the LB  
**Add**:  
```go
// Wait for LB deletion to propagate
ginkgo.By("Waiting for LB deletion to complete")
err = elbSvc.WaitUntilLoadBalancerDeleted(&elb.DescribeLoadBalancersInput{
    LoadBalancerNames: []*string{aws.String(oldLBName)},
})
if err != nil {
    log.Printf("LB deletion wait failed: %v", err)
}
```
