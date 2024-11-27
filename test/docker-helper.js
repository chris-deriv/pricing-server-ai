const { execSync } = require('child_process');
const { promisify } = require('util');
const path = require('path');
const sleep = promisify(setTimeout);

// Get project root directory (one level up from test directory)
const PROJECT_ROOT = path.resolve(__dirname, '..');
const DOCKER_CMD = `docker compose -f ${PROJECT_ROOT}/docker-compose.yml`;

async function ensureServicesRunning() {
    try {
        // Check if services are running
        const ps = execSync(`${DOCKER_CMD} ps --format json`, {
            stdio: ['pipe', 'pipe', 'pipe'],
            cwd: PROJECT_ROOT
        }).toString();

        // If not running or partially running, start them
        if (!ps.includes('contracts-service') || !ps.includes('storage-service')) {
            console.log('Starting services...');
            execSync(`${DOCKER_CMD} up -d`, {
                stdio: 'inherit',
                cwd: PROJECT_ROOT
            });
            await sleep(10000); // Wait longer for initial startup
        }
    } catch (error) {
        console.error('Failed to ensure services are running:', error);
        throw error;
    }
}

async function restartService(service) {
    try {
        await ensureServicesRunning();
        execSync(`${DOCKER_CMD} restart ${service}`, { 
            stdio: 'inherit',
            cwd: PROJECT_ROOT
        });
        // Wait for service to be ready
        await sleep(5000);
    } catch (error) {
        console.error('Failed to restart service:', error);
        throw error;
    }
}

async function restartServices(services = ['contracts-service', 'storage-service']) {
    try {
        await ensureServicesRunning();
        execSync(`${DOCKER_CMD} restart ${services.join(' ')}`, { 
            stdio: 'inherit',
            cwd: PROJECT_ROOT
        });
        // Wait for services to be ready
        await sleep(5000);
    } catch (error) {
        console.error('Failed to restart services:', error);
        throw error;
    }
}

module.exports = {
    ensureServicesRunning,
    restartService,
    restartServices
};
