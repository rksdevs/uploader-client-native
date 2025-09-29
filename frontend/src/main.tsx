import React from "react";
import { createRoot } from "react-dom/client";
import "./style.css";
import App from "./App";
import { Toaster } from "sonner";

const container = document.getElementById("root");
const root = createRoot(container!);

root.render(
  <React.StrictMode>
    <Toaster richColors />
    <App />
  </React.StrictMode>
);
