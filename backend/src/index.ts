import express from 'express';
import bodyParser from 'body-parser';

const app = express();
const port = process.env.PORT || 3001;

app.use(bodyParser.json());

app.get('/', (req, res) => {
  res.send('Hello World!');
});

app.listen(port, () => {
  console.log(`Backend server is running on http://localhost:${port}`);
});