const WebSocket = require('ws');

const ws = new WebSocket('ws://localhost:8080/ws');

ws.on('open', () => {
    console.log('WebSocket connection opened');

    const momentumMsg = 
    JSON.stringify({
        "type": "ContractSubmission",
        "data": {
          "productType": "MomentumCatcher",
          "targetMovement": 5.0,
          "duration": 60000,
          "payoff": 100.0
        }
      });

    const luckyLadderMsg = JSON.stringify({"type": "ContractSubmission","data": {"productType": "LuckyLadder","rungs": [105, 110, 115],"duration": 60000,"payoff": 100}});
    ws.send(luckyLadderMsg);
});

ws.on('message', (data) => {
    // console.log('Received:', data);
    // const priceUpdate = JSON.parse(data);
    console.log(data.toString())
    // console.log('Price:', priceUpdate.price, 'Timestamp:', priceUpdate.timestamp);
});

ws.on('error', (error) => {
    console.error('WebSocket error:', error);
});

ws.on('close', () => {
    console.log('WebSocket connection closed');
});



