#include <iostream>
#include "kasper/core/core/core.h"
#include "kasper/shell/actions/hello.h"

using namespace std;

int main() {

    printf("starting kasper node...\n");

    auto core = new Core();

    service_hello::installService(core);

    core->run();

    sleep(10);
    
    return 0;
}
