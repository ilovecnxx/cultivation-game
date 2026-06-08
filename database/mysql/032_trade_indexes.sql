-- ===================================================================
-- 交易系统索引优化 - 添加复合索引优化查询性能
-- ===================================================================

-- trade_listings 常用查询：按状态和过期时间筛选活跃挂单
ALTER TABLE trade_listings ADD INDEX idx_status_expires (status, expires_at) COMMENT '优化活跃挂单查询（WHERE status=active AND expires_at > NOW()）';

-- trade_listings 优化卖家查询 + 状态过滤
ALTER TABLE trade_listings ADD INDEX idx_seller_status (seller_id, status) COMMENT '优化卖家查询自己的挂单';

-- trade_listings 优化物品搜索（按物品ID和状态查询）
ALTER TABLE trade_listings ADD INDEX idx_item_status (item_id, status) COMMENT '优化按物品类型搜索活跃挂单';

-- trade_transactions 优化买家/卖家历史查询
ALTER TABLE trade_transactions ADD INDEX idx_buyer_created (buyer_id, created_at) COMMENT '优化买家交易历史查询';
ALTER TABLE trade_transactions ADD INDEX idx_seller_created (seller_id, created_at) COMMENT '优化卖家交易历史查询';

-- trade_auctions 优化按状态和结束时间查询（用于拍卖结算扫描）
ALTER TABLE trade_auctions ADD INDEX idx_status_endtime (status, end_time) COMMENT '优化拍卖结算查询（WHERE status=active AND end_time < NOW()）';
