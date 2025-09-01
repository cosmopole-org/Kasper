import React from "react";
import ReactDOM from "react-dom/client";
import { BrowserRouter } from "react-router-dom";
import App from "./App.tsx";
import { Provider } from "./provider.tsx";
import "@/styles/globals.css";
import Api from "./api/index.ts";
import { Actions } from "./api/client/states.tsx";
import Decillion from "./api/sigma/api/index.ts";

window.addEventListener('error', e => {
  if (e.message === 'ResizeObserver loop limit exceeded') {
    const resizeObserverErrDiv = document.getElementById(
      'webpack-dev-server-client-overlay-div'
    );
    const resizeObserverErr = document.getElementById(
      'webpack-dev-server-client-overlay'
    );
    if (resizeObserverErr) {
      resizeObserverErr.setAttribute('style', 'display: none');
    }
    if (resizeObserverErrDiv) {
      resizeObserverErrDiv.setAttribute('style', 'display: none');
    }
  }
});

let al = window.alert;

window.alert = (...params) => {
  al(...params);
  if (window.location.pathname === "/?mode=standalone") {
    setTimeout(() => {
      document.body.requestFullscreen();
    }, 100);
  }
}

export let api: Api;
Api.load().then(a => {
  api = a;
  const loadauth = () => {
    setTimeout(() => {
      Actions.updateAuthStep("auth");
    }, 2000);
  }
  if (api.services?.myPrivateKey()) {
      let authenticated = true;
      Actions.updateAuthenticated(authenticated);
      if (!authenticated) loadauth();
      else {
        Actions.updateAuthStep("passed");
      }
  } else {
    Actions.updateAuthenticated(false);
    loadauth();
  }
  ReactDOM.createRoot(document.getElementById("root")!).render(
    <React.StrictMode>
      <BrowserRouter>
        <Provider>
          <App />
        </Provider>
      </BrowserRouter>
    </React.StrictMode>,
  );
})
