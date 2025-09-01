import Decillion from "./sigma/api";
import Storage from "./sigma/helpers/storage";

class Api {
    public static async load(): Promise<Api> {
        let a = new Api();
        a.store = new Storage();
        await a.store.run();
        a.services = new Decillion(a.store);
        await a.services.connect();
        await a.loadData();
        return a;
    }
    public services?: Decillion = undefined
    public store?: Storage = undefined
    
    async loadData() {
        try {

        } catch (ex) {
            console.log(ex);
        }
    }
}

export default Api;
