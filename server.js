const express = require("express");
require("dotenv").config();
const randomstring = require("randomstring");
const fs = require("fs");
const app = express();
// https://expressjs.com/en/guide/behind-proxies.html
app.set('trust proxy', 'loopback, uniquelocal')
const port = process.env.PORT || 3000;

app.use(express.json());

let data = [];
let sharedKey;

// Create a Shared Key
if (!process.env.SHARED_KEY) {
  sharedKey = randomstring.generate();
  fs.writeFile(".env", `SHARED_KEY=${sharedKey}`, function (err) {
    if (err) return console.log(err);
    console.log(
      `Shared Key set: ${sharedKey}\nDelete .env file to generate a new key.`
    );
  });
} else {
  sharedKey = process.env.SHARED_KEY;
  console.log(
    `Shared Key: ${sharedKey}\nDelete .env file to generate a new key.`
  );
}

app.get("/health", (req, res) => {
  res.sendStatus(200);
});


app.use(function (req, res, next) {
  if (!req.headers.authorization) {
    console.log(`${req.ip} No credentials sent`)
    return res.status(401).json({
        error: "No credentials sent!",
    });
}
if (req.headers.authorization != sharedKey) {
    console.log(`${req.ip} Wrong credentials`)
    return res.status(403).json({
      error: "Wrong credentials",
    });
  }
  next();
});

// Gives Location Data
app.get("/", (req, res) => {
  let timestamp = Date.now();

  checkData(timestamp);

  res.send(data);
  console.log(`${req.ip} Data requested`)
});

// Receives Location Data
app.post("/post", (req, res) => {
  let newObj = req.body;
  let timestamp = Date.now();

  updateData(newObj, timestamp);
  checkData(timestamp);
  res.send(data);
  console.log(`${req.ip} Data received: ${JSON.stringify(req.body)}`)
});

// Updates Location Data
function updateData(newObj, timestamp) {
  let objectAlreadyExisted = false;
  data.forEach((d) => {
    if (d.name == newObj.name) {
      objectAlreadyExisted = true;
      d.name = newObj.name;
      d.x = newObj.waypoint.x;
      d.y = newObj.waypoint.y;
      d.plane = newObj.waypoint.plane;
      d.type = newObj.type;
      d.title = newObj.title;
      d.world = newObj.world;
      d.timestamp = timestamp;
    }
  });

  if (!objectAlreadyExisted)
    data.push({
      name: newObj.name,
      x: newObj.waypoint.x,
      y: newObj.waypoint.y,
      plane: newObj.waypoint.plane,
      type: newObj.type,
      title: newObj.title,
      world: newObj.world,
      timestamp: timestamp,
    });
}

// Checks if it has received data from a user for the last 5 seconds
function checkData(timestamp) {
  for (let i = data.length - 1; i >= 0; i--) {
    if (timestamp - data[i].timestamp > 5000) {
      data.splice(i);
    }
  }
}

app.listen(port, () => {
  console.log(`Friend tracker listening at http://localhost:${port}`);
});
