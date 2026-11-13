import { createRoot } from "react-dom/client";
import "./index.css";
import Overlay from "./Overlay";
import SettingsWindow from "./Settings";

const overlay = location.hash.startsWith("#/overlay");
document.documentElement.classList.toggle("overlay", overlay);
createRoot(document.getElementById("root")!).render(overlay ? <Overlay /> : <SettingsWindow />);
