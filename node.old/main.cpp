#include <iostream>
#include <condition_variable>
#include <mutex>
#include "kasper/core/core/core.h"
#include "kasper/shell/actions/hello/hello.h"
#include "kasper/shell/actions/user/user.h"
#include "kasper/shell/actions/point/point.h"

using namespace std;

int main()
{

    std::cerr << "starting kasper node..." << std::endl;

    auto core = new Core();

    service_hello::installService(core);
    service_user::installService(core);
    service_point::installService(core);

    core->run();

    std::condition_variable cv;
    std::mutex m;
    std::unique_lock<std::mutex> lock(m);
    cv.wait(lock, []
            { return false; });

    return 0;
}
