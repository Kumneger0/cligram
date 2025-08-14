
import pino from 'pino';
import * as rfs from "rotating-file-stream";

const stream = rfs.createStream("cligram-js-backend.log", {
    interval: "1d",
    path: "/tmp",
    maxFiles: 7,
});

const logger = pino({}, stream);
export default logger;
