const WebSocket = require('ws');

const ws = new WebSocket('ws://localhost:8080/ws');

ws.on('open', () => {
    console.log('WebSocket connection opened');

    const momentumMsg = JSON.stringify({
        "type": "ContractSubmission",
        "data": {
            "productType": "MomentumCatcher",
            "targetMovement": 5.0,
            "duration": 60000,
            "payoff": 100.0
        }
    });

    const luckyLadderMsg = JSON.stringify({
        "type": "ContractSubmission",
        "data": {
            "productType": "LuckyLadder",
            "rungs": [105, 110, 115],
            "duration": 6000,
            "payoff": 100
        }
    });

    // Uncomment one of these to test different contract types
    ws.send(luckyLadderMsg);
    //ws.send(momentumMsg);
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
            console.log(update);
            // console.log('Contract Update:');
            // console.log('- Contract ID:', update.contractID);
            // console.log('- Status:', update.status);
            // console.log('- Price:', update.price);
            // console.log('- Timestamp:', update.timestamp);

            // MomentumCatcher specific fields
            // if (update.movement !== undefined) {
            //     console.log('- Movement:', update.movement);
            //     console.log('- Target Hit:', update.target_hit);
            // }

            // LuckyLadder specific fields
            // if (update.rungs_hit) {
            //     console.log('- Rungs Hit:', update.rungs_hit);
            // }
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
