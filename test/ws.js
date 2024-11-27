const WebSocket = require('ws');

const ws = new WebSocket('ws://localhost:8080/ws');

ws.on('open', () => {
    console.log('WebSocket connection opened');

    const momentumDuration = 2000;  // 2 seconds
    const ladderDuration = 5000;   // 60 seconds

    console.log(`Creating MomentumCatcher with duration: ${momentumDuration}ms`);
    const momentumMsg = JSON.stringify({
        "type": "ContractSubmission",
        "data": {
            "productType": "MomentumCatcher",
            "targetMovement": 5.0,
            "duration": momentumDuration,
            "payoff": 100.0
        }
    });

    console.log(`Creating LuckyLadder with duration: ${ladderDuration}ms`);
    const luckyLadderMsg = JSON.stringify({
        "type": "ContractSubmission",
        "data": {
            "productType": "LuckyLadder",
            "rungs": [105, 110, 115],
            "duration": ladderDuration,
            "payoff": 100
        }
    });

    // Send both contracts with a slight delay to ensure they're processed separately
    ws.send(luckyLadderMsg);
    setTimeout(() => {
        ws.send(momentumMsg);
    }, 100);
});

ws.on('message', (data) => {
    const message = JSON.parse(data.toString());
    console.log('\nReceived message type:', message.type);

    switch (message.type) {
        case 'ContractAccepted':
            console.log('Contract accepted with ID:', message.contractID);
            break;

        case 'ContractUpdate':
            const update = message.data;
            // Add timestamp to see when updates are received
            console.log(`${new Date().toISOString()} Contract Update:`, update);
            break;

        case 'Error':
            console.error('Error:', message.message);
            break;

        default:
            console.log('Unknown message type:', message);
    }
});

ws.on('error', (error) => {
    console.error('WebSocket error:', error);
});

ws.on('close', () => {
    console.log('WebSocket connection closed');
});
