// 格式化字节单位
export const formatBytes = (bytes) => {
    bytes = Number(bytes) || 0;
    const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB'];
    let i = 0;
    while (bytes >= 1024 && i < units.length - 1) {
        bytes /= 1024;
        i++;
    }
    return bytes.toFixed(2) + ' ' + units[i];
}

// 计算剩余天数的函数
export const calculateRemainingDays = (expireTime) => {
    if (!expireTime) {
        return '-'
    }
    const normalizedTime = Number(expireTime) > 0 && Number(expireTime) < 1000000000000 ? Number(expireTime) * 1000 : expireTime
    const expireDate = new Date(normalizedTime)
    const today = new Date()
    const diffTime = expireDate - today
    return Math.ceil(diffTime / (1000 * 60 * 60 * 24)) + '天'
}

export const formatBandwithBytes = (bytes) => {
    bytes = bytes * 8;
    const units = ['bps', 'Kbps', 'Mbps', 'Gbps', 'Tbps', 'Pbps', 'Ebps', 'Zbps', 'Ybps'];
    let i = 0;
    while (bytes >= 1024 && i < units.length - 1) {
        bytes /= 1024;
        i++;
    }
    return (bytes).toFixed(2) + ' ' + units[i];
}

// 格式化系统时间为小时:分钟:秒
export const formatTime = (seconds) => {
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    const remainingSeconds = seconds % 60;
    return `${hours}小时 ${minutes}分钟 ${remainingSeconds}秒`;
}

// 格式化时间戳为可读的日期格式
export const formatTimeStamp = (timestamp) => {
    const date = new Date(timestamp * 1000); // 转换为毫秒
    const year = date.getFullYear();
    const month = ('0' + (date.getMonth() + 1)).slice(-2);
    const day = ('0' + date.getDate()).slice(-2);
    const hours = ('0' + date.getHours()).slice(-2);
    const minutes = ('0' + date.getMinutes()).slice(-2);
    const seconds = ('0' + date.getSeconds()).slice(-2);
    return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`;
}

export const formatUptime = (seconds) => {
    seconds = Number(seconds) || 0;
    const days = Math.floor(seconds / (24 * 3600));
    seconds %= 24 * 3600;
    const hours = Math.floor(seconds / 3600);
    seconds %= 3600;
    const minutes = Math.floor(seconds / 60);
    const secs = seconds % 60;

    return `${days}d ${hours}h ${minutes}m ${secs}s`;
}

export const formatUptimeZh = (seconds) => {
    seconds = Number(seconds) || 0;
    const days = Math.floor(seconds / (24 * 3600));
    seconds %= 24 * 3600;
    const hours = Math.floor(seconds / 3600);
    seconds %= 3600;
    const minutes = Math.floor(seconds / 60);
    if (days > 0) return `${days}天 ${hours}小时`;
    if (hours > 0) return `${hours}小时 ${minutes}分钟`;
    return `${minutes}分钟`;
}

export const formatAgo = (timestamp) => {
    if (!timestamp) return '-';
    const diff = Math.max(0, Math.floor(Date.now() / 1000) - Number(timestamp));
    if (diff < 60) return `${diff} 秒前`;
    if (diff < 3600) return `${Math.floor(diff / 60)} 分钟前`;
    if (diff < 86400) return `${Math.floor(diff / 3600)} 小时前`;
    return `${Math.floor(diff / 86400)} 天前`;
}
