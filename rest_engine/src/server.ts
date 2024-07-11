import http from "http";
import { app } from "./app";

const server = http.createServer(app).listen(3000);
console.log("Web API started.");

export { server };
