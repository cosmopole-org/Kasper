#include <iostream>
#include "kasper/core/core/core.h"
#include "kasper/shell/actions/hello/hello.h"
#include "kasper/shell/actions/user/user.h"
#include "kasper/shell/actions/point/point.h"

using namespace std;

int main() {

    printf("starting kasper node...\n");

    auto core = new Core();

    service_hello::installService(core);
    service_user::installService(core);
    service_point::installService(core);

    core->run();

    sleep(10);
    
    return 0;
}
