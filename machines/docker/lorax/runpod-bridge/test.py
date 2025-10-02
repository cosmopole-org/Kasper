import requests
import json
import time

def test():

    lockId = "9c027cb0-d87f-498e-b4f8-148944e4976c-d7f01e7b-02cb-49e7-847a-97c61dc0a039"
    signature = "VTs+NizLIrBwMjXwRrSXwFndaB3f+PE+iBuyg0B0qkRJ79pOZfU1PtRnOfnHu0huLUi+uu/5FbzLW4xvwghIJSxzhGA8XdEnuuMBQIAvEyBHB/kZ77HWcabhCLCRd6uaFtE/pDfBCOww4Dc1CwC4/jZ6z0UoE9fSVtknmLblIQX/RV0WpkPow44TSKVBp+9rVJ3qp+KOXcKYFUs5MJI8vskP2kwHkxylWKPRn/JumjmX9osMQCNGoIsUi/l8Q8+ILwnVbtXTB5VJk1HyC3cgbPtdw/3ztJbUXkG81cCtcB5+QnNwJ4HIwl7ibWnqATjYtIbKvVogMuJ1HDqaZVzCmFd99BVljg4HxPI6deLZQcvQgsBqw7KZC24NTJT0lmALOrYIdEnnURLWG1kAcHvi5zpHWIRal9y9HS2kRpermp57cwziqNvTHCnmSHE7rbdeCcU5+1ZKAFUABfws2qsx7uJKg5UpuEt/LI4peIi5DkvJrxCcWmw/fIIDRYss+2zVX5NeUqLiMiHCVeOHCJ5hv4Coe+wIKqq81nje3PjQ0QQOZ0K8QBy/AsXGkcjNVM+7NDKKLBgSGdyBvaMtDzqU+L5NeRxs8LCTKsLPDZTL1x0te0jiqwnzvOvzXO2xplEhBCo5n3Ov83QhGYNrvglwxyEoZ3OxJHG0lbipf4HjOxY="
    userId = "8@global"

    # Wait for the chain bridge to be ready
    print("Waiting for chain bridge to start...")
    max_retries = 60
    for i in range(max_retries):
        try:
            response = requests.get('http://localhost:3000/health')
            if response.status_code == 200:
                print("chain bridge is ready!")
                break
        except Exception as e:
            print('An exception occurred: {}'.format(e))
            pass

        time.sleep(10)
        print(f"Waiting for bridge to start... ({i+1}/{max_retries})")

    url = "http://localhost:3000/consumeLock"
    payload = json.dumps({
        "lockId": lockId,
        "signature": signature,
        "userId": userId
    })
    headers = {
        'Content-Type': 'application/json',
    }
    response = requests.request("POST", url, headers=headers, data=payload)
    if response.json()["success"] is not True:
        return {"error": "payment consumption has failed " + str(response.status_code) + " " + response.text}
    else:
        return {"success": True}

print(test())