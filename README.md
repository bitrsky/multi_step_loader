> 在业务开发过程中，经常会遇到一种情况：
某个接口需要某个组合数据，但是该数据不可能从某个地方直接获取到，而是要从多个地方，分阶段拉取多次，最后才能组合成需要的数据。


#### 1. 一个例子
    小明同学最近接到一个需求，需要给前端提供一个接口A。
    接口A:
    * 参数: 文章id
    * 返回：
        - 文章摘要，封面图
        - 文章评论数
        - 文章点赞数
        - 文章作者信息
        - 文章作者粉丝数

    看到这个需求，小明细细想了一下，so easy，随即拆为以下几个步骤处理：
    - 1 首先通过文章id去获取文章数据（包含文章摘要，封面图，评论Id，作者Id）
    - 2 通过文章ID获取最新的点赞数
    - 3 通过评论ID获取最新的评论数
    - 4 通过作者ID获取作者信息
    - 5 通过作者ID获取粉丝数
    
    作为一个工作1年的程序猿，这个逻辑完全不在话下，小明同学三下五初二就把逻辑搞定，部署测试了一下，完全没问题。
    
    当把代码提交到小肖同学review时，小肖同学非常老道的指出：第1,2步完全可以并行执行，第3,4,5步也可以并行执行，这样可以大大减低接口耗时。
    
    小明同学恍然大悟，添加了并行逻辑，将1,2同时执行，3,4,5步同时执行，然后部署测试，果然耗时减低了不少，小肖不愧为大佬。高高兴兴的再一次提交了代码，让肖帮忙review。
    
    肖认真阅读后，发现代码库中好多地方充斥着这种逻辑，每次接口都需要重新定义好多结构，go func() 逻辑到处存在，所以语重心长的和小明说：这种通过多个阶段获取不同的数据进而组合的逻辑，你也遇到不少了，说明这是一个很常见的逻辑，你试着把它抽象出一个模型试试。
    
    小明陷入了长长的思考。。。。。

    
#### 2. 模型抽象

![image](https://note.youdao.com/yws/res/319/5539C2C05AED4804AAE11D5E472EF4AD)
    
步骤1：
-  协程1：通过文章ID获取文章基本信息，包含摘要，封面图，作者ID，评论ID
-  协程2：通过文章ID获取点赞数

步骤2：
- 协程1：通过作者ID获取作者信息
- 协程2：通过作者ID获取粉丝数
- 协程3：通过评论ID获取评论数


如果把数据的一次获取过程定义为一次loader，每次loader的数据需要存放到某个Item中，并且所有loader能够继续访问这个Item。

定义全局Item结构和loader接口

```Golang
// Item
type Item struct {
	ArticleID []string // 文章ID

	ArticleInfo map[string]*Article // 文章ID->基本信息
	ArticleUps  map[string]int      // 文章ID->点赞数

	AuthorID   []string           // 作者ID
	AuthorInfo map[string]*Author // 作者ID->基本信息
	Fans       map[string]int     // 作者ID->粉丝数

	CommentID  []string       // 评论ID
	CommentNum map[string]int // 评论ID->评论数
}


```

```golang
// Loader interface
type Loader interface {
	// 开始加载数据
	StartLoadData(context.Context, []*model.Item) error

	// 把加载的数据赋值给Item,并准备后一步骤需要的数据
	SetDataToItems(context.Context, []*model.Item) error

	// 判断loader是否执行完成
	IsReady() bool

	// 获取loader命名
	Name() string
}
```


定义以下loader的实现：
- ArticleInfoLoader: 文章内容获取
- AuthorInfoLoader: 作者信息获取
- FansNumLoader: 粉丝数获取
- CommentNumLoader: 评论数获取
- ArticleUpsLoader: 点赞数获取

ArticleInfoLoader，AuthorInfoLoader，FansNumLoader，CommentNumLoader，ArticleUpsLoader等都是loader的实现，具体实现如下：以ArticleInfoLoader为例

``` Golang
type ArticleInfoLoader struct {
	articleInfo map[string]*model.Article
	isReady     bool
}

// NewArticleInfoLoader 创建ArticleInfoLoader实例
func NewArticleInfoLoader() *ArticleInfoLoader {
	return &ArticleInfoLoader{
		articleInfo: make(map[string]*model.User),
	}
}

// StartLoadData load数据
func (l *ArticleInfoLoader) StartLoadData(ctx context.Context, items []*model.Item) (err error) {
	if len(items) == 0 {
		return nil
	}

	articleIDs := []string{}
	for _, item := range items {
		articleIDs = append(articleIDs, item.articleIDs...)
	}

	if len(articleIDs) <= 0 {
		return nil
	}

	articleInfo, err := getArticleInfo() // 底层获取文章信息封装接口
	if err != nil {
		return err
	}

	l.articleInfo = articleInfo
	l.setReady()
	return nil
}

// SetDataToItems 将拉取到的数据存放到item中，并将下一阶段需要的数据也放进去
func (l *ArticleInfoLoader) SetDataToItems(ctx context.Context, items []*model.Item) error {
	for _, item := range items {
		for _, articleID := range item.articleIDs() {
			if articleInfo, ok := l.articleInfo[articleID]; ok {
				item.PutArticleInfo(articleID, articleInfo)
				item.AddAuthorID(articleInfo.AuthorID)
				item.AddCommentID(articleInfo.CommentID)
			}
		}
	}
	return nil
}

func (l *ArticleInfoLoader) setReady() {
	l.isReady = true
}

func (l *ArticleInfoLoader) IsReady() bool {
	return l.isReady
}

func (l *ArticleInfoLoader) Name() string {
	return "ArticleInfoLoader"
}

```

定义好需要的结构后，上述例子就可以抽象为两步，第一步并行执行ArticleInfoLoader和ArticleUpsLoader，第二步并行执行AuthorInfoLoader，CommentNumLoader和FansNumLoader。

接下来我们需要定义一个loader的管理对象，定义为loaderManager，loaderManager的作用就是按照用户定义好的步骤执行loader，每一步并行执行多个loader。

``` Golang
type LoadManager struct {
	parallelLoaders []*ParallelLoaders
}

func NewLoadManager(parallelLoaders ...*ParallelLoaders) *LoadManager {
	return &LoadManager{
		parallelLoaders: parallelLoaders,
	}
}

func (loadmgr *LoadManager) LoadData(ctx context.Context, items []*model.Item) error {
	for _, pLoaders := range loadmgr.parallelLoaders {
		pLoaders.LoadItemsData(ctx, items)
	}
	return nil
}

type ParallelLoaders struct {
	loaders []Loader
	timeout time.Duration
}

func NewParallelLoaders(timeout time.Duration, loaders ...Loader) *ParallelLoaders {
	return &ParallelLoaders{
		loaders: loaders,
		timeout: timeout,
	}
}

// 增加并发loader列表中的loader
func (pLoaders *ParallelLoaders) AppendLoader(loader Loader) error {
	if loader != nil {
		pLoaders.loaders = append(pLoaders.loaders, loader)
	}
	return nil
}

// 并发调用loader
func (pLoaders *ParallelLoaders) LoadItemsData(ctx context.Context, items []*model.Item) error {
	var wg sync.WaitGroup
	for _, loader := range pLoaders.loaders {
		wg.Add(1)
		go func(loader Loader) {
			defer wg.Done()
			loadWithTimeout(ctx, loader, pLoaders.timeout, items)
		}(loader)
	}
	wg.Wait()
	for _, loader := range pLoaders.loaders {
		if loader.IsReady() {
			loader.SetDataToItems(ctx, items)
		}
	}
	return nil
}

func loadWithTimeout(ctx context.Context, loader Loader, timeout time.Duration, items []*model.Item) (err error) {

	var (
		errCh = make(chan error, 1)
	)

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	go func() {
		errCh <- loader.StartLoadData(ctx, items)
	}()

	select {
	case err = <-errCh:
	case <-ctx.Done():
		err = ctx.Err()
	}

	if err != nil {
		log.Errorf("loader=%s err=%s", loader.Name(), err.Error())
	}
	return err
}

```

设计好loaderManage之后，上面的获取内容的问题就简单多了，只需要写下面几句代码就可以拉到所有的数据了

```
func loaderData(articleIDs []string) {
	item := &model.Item{
		ArticleIDs: articleIDs,
	}
	items := &[]*model.Item{item}
	parallelLoader0 := loader.NewParallelLoaders(time.Millisecond * 500)
	parallelLoader0.AppendLoader(loader.NewArticleInfoLoader())
	parallelLoader0.AppendLoader(loader.NewArticleUpsLoader())

	parallelLoader1 := loader.NewParallelLoaders(time.Millisecond * 500)
	parallelLoader1.AppendLoader(loader.NewCommentNumLoader())
	parallelLoader1.AppendLoader(loader.NewAuthorInfoLoader())
	parallelLoader1.AppendLoader(loader.NewFansNumLoader())

	loaderMgr := loader.NewLoadManager(parallelLoader0, parallelLoader1)
	loaderMgr.LoadData(context.TODO(), items)
}

```
执行完之后，组装数据所需要的信息就都存在于对象items中了。


#### 3. 模型优化

考虑下面的例子，整个流程包含A、B、C、D、E等5个loader，每个loader的依赖关系和执行时间如图。

![image](https://note.youdao.com/yws/res/1011/A03DCFCEBB1A4C9FBB8C1F86F0CBF367)

如果按照上面的执行逻辑，这个例子需要分三步进行，第一步执行A loader，耗时5，第二步执行B和Cloader，耗时40，第三步执行D和E loader，耗时30，整个流程需要耗时至少75。

通过观察依赖关系，D在B执行完成就可以执行了，不需要等待C执行完，所以整个流程最短执行时间可以缩短到55。

优化后的loaderManager代码如下：
```golang

// LoadManager loader管理器
type LoadManager struct {
	parallelLoaders []*ParallelLoaders
}

// NewLoadManager 构造loader管理器
func NewLoadManager(parallelLoaders ...*ParallelLoaders) *LoadManager {
	return &LoadManager{
		parallelLoaders: parallelLoaders,
	}
}

func (loadmgr *LoadManager) LoadData(ctx context.Context, items []*model.Item) error {

	for _, pLoaders := range loadmgr.parallelLoaders {
		pLoaders.LoadItemsData(ctx, items)
	}
	return nil
}

type LoaderDataInterface interface {
	// 开始加载数据
	StartLoadData(context.Context, []*model.Item) error
	// 把加载的数据赋值给Item
	SetDataToItems(context.Context, []*model.Item) error
	// 数据是否加载完成
	IsReady() bool
	// Loader 的名字
	Name() string
}

// ParallelLoaders 同一批次loader
type ParallelLoaders struct {
	loaders []LoaderDataInterface
	timeout time.Duration
}

// NewParallelLoaders 构造并发loader
func NewParallelLoaders(timeout time.Duration, loaders ...LoaderDataInterface) *ParallelLoaders {

	return &ParallelLoaders{
		loaders: loaders,
		timeout: timeout,
	}
}

// AppendLoader 增加并发loader列表中的loader
func (pLoaders *ParallelLoaders) AppendLoader(loader LoaderDataInterface) error {

	if loader != nil {
		pLoaders.loaders = append(pLoaders.loaders, loader)
	}

	return nil
}

// LoadItemsData 并发调用loader，获取文章等数据，并设置item中的属性
func (pLoaders *ParallelLoaders) LoadItemsData(ctx context.Context, items []*model.Item) error {

	var wg sync.WaitGroup
	// 并发执行同一批次loader
	for _, loader := range pLoaders.loaders {
		wg.Add(1)
		go func(loader LoaderDataInterface) {
			defer wg.Done()
			loadWithTimeout(ctx, loader, pLoaders.timeout, items)
		}(loader)
	}

	wg.Wait()
	// 串行回写数据到item中，避免并发读写map
	for _, loader := range pLoaders.loaders {
		if loader.IsReady() {
			loader.SetDataToItems(ctx, items)
		}
	}
	return nil
}

// loadWithTimeout loader超时控制
func loadWithTimeout(ctx context.Context, loader LoaderDataInterface,
	timeout time.Duration, items []*model.Item) (err error) {

	var (
		errCh = make(chan error, 1)
	)

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	go func() {
		errCh <- loader.StartLoadData(ctx, items)
	}()

	select {
	case err = <-errCh:
	case <-ctx.Done():
		err = ctx.Err()
	}

	if err != nil {
		log.Errorf("loader=%s err=%s", loader.Name(), err.Error())
	}
	return err
}

```

这样整个流程的构建代码如下：
```golang
    loader5 := loader.NewWaitLoader(5)
	loader10 := loader.NewWaitLoader(10)
	loader20 := loader.NewWaitLoader(20)
	loader30 := loader.NewWaitLoader(30)
	loader40 := loader.NewWaitLoader(40)

	loaderManager := core.NewLoaderManager(100)
	loaderManager.AddLoaders(loader5, loader10, loader20, loader30, loader40)

	loaderManager.Link(loader5, loader20, loader40)
	loaderManager.Link(loader20, loader30)
	loaderManager.Link(loader40, loader10)
	fmt.Println("flow start:", time.Now().Nanosecond()/1000000)
	if err := loaderManager.LoadData(context.Background(), []*model.Item{}); err != nil {
		fmt.Println(err)
	}
```

#### 4. 模型优点

当项目中的所有获取数据逻辑都使用这种loader逻辑封装

优点：
- 增加项目的可维护性
- 增加代码的可读性，开发人员只需要了解每个loader的功能就可以，减少同一个逻辑在多出出现的风险
- 可统一添加错误报警机制，
- 非常方便解决了业务逻辑多阶段从多处获取数据的问题

缺点：

- 如果只需要从一个地方获取数据，也需要构建loader的整个过程，比较麻烦



#### 5 代码库
github：https://github.com/BitrSky/multi_step_loader
