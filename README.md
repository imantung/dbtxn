# DBTXN

Handling Database Transaction in Logic layer

## Usage 

In `Repository`/Data Access layer
```go
func (r *RepoImpl) Delete(ctx context.Context) (int64, error) {
  txn, err := dbtxn.Use(ctx, r.DB) // use transaction if begin detected
  if err != nil {                  // create transaction error
      return -1, err
  }

  db := txn                     // transaction object or database connection

  // result, err := ...

  if err != nil {
      txn.AppendError(err)            // append error to plan for rollback
      return -1, err
  }
  // ...
}
```

In `Service`/Business Logic layer
```go
func (s *SvcImpl) SomeOperation(ctx context.Context) (err error){
  // begin the transaction
  txn := dbtxn.Begin(&ctx)

  // commit/rollback in end function
  defer func(){ err = txn.Commit() }()
  
  // ...
}
```


## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details